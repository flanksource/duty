package tests

import (
	gocontext "context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/postq/pg"
)

var _ = ginkgo.Describe("people table activity", func() {
	ginkgo.It("notifies on creation, soft deletion and deletion, not on other updates", func() {
		activity := make(chan string, 100)
		ctx, cancel := gocontext.WithCancel(DefaultContext)
		defer cancel()

		go func() {
			defer ginkgo.GinkgoRecover()
			_ = pg.ListenMany(context.NewContext(ctx), pg.ChannelListener{Channel: "table_activity", Receiver: activity})
		}()
		time.Sleep(100 * time.Millisecond)

		person := models.Person{ID: uuid.New(), Name: "Notify Person", Email: "notify-person@example.com"}

		// next returns the next people notification for the person, or "" when none arrives in time.
		next := func(timeout time.Duration) string {
			deadline := time.After(timeout)
			for {
				select {
				case payload := <-activity:
					if strings.HasPrefix(payload, "people ") && strings.HasSuffix(payload, person.ID.String()) {
						return payload
					}
				case <-deadline:
					return ""
				}
			}
		}

		Expect(DefaultContext.DB().Create(&person).Error).To(Succeed())
		Expect(next(2 * time.Second)).To(Equal("people INSERT " + person.ID.String()))

		Expect(DefaultContext.DB().Model(&person).Update("last_login", time.Now()).Error).To(Succeed())
		Expect(DefaultContext.DB().Model(&person).Update("name", "Renamed Notify Person").Error).To(Succeed())
		Expect(next(500 * time.Millisecond)).To(BeEmpty())

		Expect(DefaultContext.DB().Model(&person).Update("deleted_at", time.Now()).Error).To(Succeed())
		Expect(next(2 * time.Second)).To(Equal("people UPDATE " + person.ID.String()))

		Expect(DefaultContext.DB().Delete(&person).Error).To(Succeed())
		Expect(next(2 * time.Second)).To(Equal("people DELETE " + person.ID.String()))
	})
})
