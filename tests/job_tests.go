package tests

import (
	"sync/atomic"
	"time"

	"github.com/flanksource/duty"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Job", Ordered, func() {
	var ctx context.Context
	var sampleJob *job.Job
	It("Prevent concurrent execution", func() {
		var counter = atomic.Int32{}
		ctx = testutils.DefaultContext
		sampleJob = &job.Job{
			Name:       "test",
			Singleton:  true,
			JobHistory: true,
			Context:    ctx,
			Fn: func(ctx job.JobRuntime) error {
				counter.Add(1)
				time.Sleep(50 * time.Millisecond)
				return nil
			},
		}
		_ = duty.UpdateProperty(ctx, "test.trace", "true")
		_ = duty.UpdateProperty(ctx, "test.db.level", "trace")

		sampleJob.Run()
		Expect(sampleJob.Retention.Success).To(Equal(3))
		Expect(sampleJob.Retention.Failed).To(Equal(3))
		Expect(sampleJob.Retention.Age).To(Equal(time.Hour * 24 * 30))
		Expect(sampleJob.Retention.Interval).To(Equal(time.Hour))

		current := counter.Load()
		go sampleJob.Run()
		go sampleJob.Run()
		go sampleJob.Run()
		time.Sleep(100 * time.Millisecond)
		Expect(counter.Load()).To(Equal(current + 1))
	})
	It("Should clean up jobs", func() {
		items, _ := sampleJob.FindHistory()

		Expect(len(items)).To(BeNumerically("==", 4))
		sampleJob.Singleton = false
		sampleJob.Run()
		sampleJob.Run()
		sampleJob.Retention.Interval = time.Millisecond
		sampleJob.Retention.Success = 2
		sampleJob.Run()

		Eventually(func() int64 {
			items, _ := sampleJob.FindHistory()
			return int64(len(items))
		}, "10s").Should(BeNumerically("==", 2))

	})
})
