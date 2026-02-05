package tests

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/postq/pg"
)

var _ = ginkgo.Describe("PGListen", func() {
	ginkgo.It("should distribute notifications to multiple listeners on same channel", func() {
		channel := fmt.Sprintf("test_channel_multiple_%d", time.Now().Unix())
		ch1 := make(chan string, 10)
		ch2 := make(chan string, 10)
		ch3 := make(chan string, 10)

		// Start listener in background
		ctx, cancel := gocontext.WithCancel(DefaultContext)
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			err := pg.ListenMany(context.NewContext(ctx),
				pg.ChannelListener{Channel: channel, Receiver: ch1},
				pg.ChannelListener{Channel: channel, Receiver: ch2},
				pg.ChannelListener{Channel: channel, Receiver: ch3},
			)
			Expect(err).To(BeNil())
		}()

		// Give the listener time to establish connection and execute LISTEN
		time.Sleep(100 * time.Millisecond)

		// Send a notification
		testPayload := fmt.Sprintf("test_payload_%d", time.Now().UnixNano())
		_, err := DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", channel, testPayload))
		Expect(err).To(BeNil())

		// All three listeners should receive the notification
		Eventually(ch1, 2*time.Second).Should(Receive(Equal(testPayload)))
		Expect(ch2).To(Receive(Equal(testPayload)))
		Expect(ch3).To(Receive(Equal(testPayload)))
	})

	ginkgo.It("should handle multiple channels on same connection", func() {
		ts := time.Now().Unix()
		channel1 := fmt.Sprintf("test_channel_1_%d", ts)
		channel2 := fmt.Sprintf("test_channel_2_%d", ts)
		channel3 := fmt.Sprintf("test_channel_3_%d", ts)

		ch1 := make(chan string, 10)
		ch2 := make(chan string, 10)
		ch3 := make(chan string, 10)

		// Start listener in background
		ctx, cancel := gocontext.WithCancel(DefaultContext)
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			err := pg.ListenMany(context.NewContext(ctx),
				pg.ChannelListener{Channel: channel1, Receiver: ch1},
				pg.ChannelListener{Channel: channel2, Receiver: ch2},
				pg.ChannelListener{Channel: channel3, Receiver: ch3},
			)
			Expect(err).To(BeNil())
		}()

		// Give the listener time to establish connection and execute LISTEN
		time.Sleep(100 * time.Millisecond)

		// Send notifications to different channels
		nano := time.Now().UnixNano()
		payload1 := fmt.Sprintf("payload_1_%d", nano)
		payload2 := fmt.Sprintf("payload_2_%d", nano)
		payload3 := fmt.Sprintf("payload_3_%d", nano)

		_, err := DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", channel1, payload1))
		Expect(err).To(BeNil())

		_, err = DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", channel2, payload2))
		Expect(err).To(BeNil())

		_, err = DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", channel3, payload3))
		Expect(err).To(BeNil())

		// Each listener should receive only its channel's notification
		Eventually(ch1, 2*time.Second).Should(Receive(Equal(payload1)))
		Eventually(ch2, 2*time.Second).Should(Receive(Equal(payload2)))
		Eventually(ch3, 2*time.Second).Should(Receive(Equal(payload3)))
	})

	ginkgo.It("should handle mixed channels - multiple listeners on some, single on others", func() {
		ts := time.Now().Unix()
		sharedChannel := fmt.Sprintf("shared_channel_%d", ts)
		uniqueChannel := fmt.Sprintf("unique_channel_%d", ts)

		// Multiple listeners on shared channel
		shared1 := make(chan string, 10)
		shared2 := make(chan string, 10)
		// Single listener on unique channel
		unique := make(chan string, 10)

		// Start listener in background
		ctx, cancel := gocontext.WithCancel(DefaultContext)
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			err := pg.ListenMany(context.NewContext(ctx),
				pg.ChannelListener{Channel: sharedChannel, Receiver: shared1},
				pg.ChannelListener{Channel: sharedChannel, Receiver: shared2},
				pg.ChannelListener{Channel: uniqueChannel, Receiver: unique},
			)
			Expect(err).To(BeNil())
		}()

		time.Sleep(100 * time.Millisecond)

		// Send notifications
		nano := time.Now().UnixNano()
		sharedPayload := fmt.Sprintf("shared_payload_%d", nano)
		uniquePayload := fmt.Sprintf("unique_payload_%d", nano)

		_, err := DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", sharedChannel, sharedPayload))
		Expect(err).To(BeNil())

		_, err = DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", uniqueChannel, uniquePayload))
		Expect(err).To(BeNil())

		// Both shared listeners should receive shared notification
		Eventually(shared1, 2*time.Second).Should(Receive(Equal(sharedPayload)))
		Eventually(shared2, 2*time.Second).Should(Receive(Equal(sharedPayload)))
		// Unique listener should receive unique notification
		Eventually(unique, 2*time.Second).Should(Receive(Equal(uniquePayload)))
	})

	ginkgo.It("should isolate channels properly", func() {
		ts := time.Now().Unix()
		channel1 := fmt.Sprintf("isolation_test_1_%d", ts)
		channel2 := fmt.Sprintf("isolation_test_2_%d", ts)

		ch1 := make(chan string, 10)
		ch2 := make(chan string, 10)

		// Start listener in background
		ctx, cancel := gocontext.WithCancel(DefaultContext)
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			err := pg.ListenMany(context.NewContext(ctx),
				pg.ChannelListener{Channel: channel1, Receiver: ch1},
				pg.ChannelListener{Channel: channel2, Receiver: ch2},
			)
			Expect(err).To(BeNil())
		}()

		time.Sleep(100 * time.Millisecond)

		// Send notification only to channel1
		payload := fmt.Sprintf("isolated_payload_%d", time.Now().UnixNano())
		_, err := DefaultContext.Pool().Exec(DefaultContext, fmt.Sprintf("NOTIFY %s, '%s'", channel1, payload))
		Expect(err).To(BeNil())

		// ch1 should receive, ch2 should not
		Eventually(ch1, 2*time.Second).Should(Receive(Equal(payload)))

		// ch2 should timeout (no notification)
		Consistently(ch2, 500*time.Millisecond).ShouldNot(Receive())
	})
})
