package tests

import (
	"github.com/flanksource/duty/api"
	"sync/atomic"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("Job", Ordered, func() {
	var ctx context.Context
	var sampleJob *job.Job

	It("Prevent concurrent execution", func() {
		var counter = atomic.Int32{}
		ctx = DefaultContext
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
		_ = context.UpdateProperty(ctx, "test.trace", "true")
		_ = context.UpdateProperty(ctx, "test.db.level", "trace")
		_ = context.UpdateProperty(ctx, api.PropertyJobEvictionPeriod, "1s")
		_ = context.UpdateProperty(ctx, api.PropertyJobJitterDisable, "true")

		sampleJob.Run()
		Expect(sampleJob.Retention.Success).To(Equal(1))
		Expect(sampleJob.Retention.Failed).To(Equal(3))

		current := counter.Load()
		go sampleJob.Run()
		go sampleJob.Run()
		go sampleJob.Run()
		time.Sleep(100 * time.Millisecond)
		Expect(counter.Load()).To(Equal(current + 1))
	})

	It("Should skip disabled jobs", func() {
		var counter = atomic.Int32{}
		disabledJob := &job.Job{
			Name:       "test-disable",
			Singleton:  true,
			JobHistory: true,
			Context:    DefaultContext,
			Fn: func(ctx job.JobRuntime) error {
				counter.Add(1)
				return nil
			},
		}

		before, err := disabledJob.FindHistory()
		Expect(err).ToNot(HaveOccurred())

		_ = context.UpdateProperty(DefaultContext, "jobs.test-disable.disable", "true")
		disabledJob.Run()
		Expect(counter.Load()).To(BeZero())

		after, err := disabledJob.FindHistory()
		Expect(err).ToNot(HaveOccurred())
		Expect(after).To(HaveLen(len(before)))
	})

	It("Should clean up jobs", func() {
		items, _ := sampleJob.FindHistory()

		groups := lo.GroupBy(items, func(j models.JobHistory) string { return j.Status })
		counts := lo.CountValuesBy(items, func(j models.JobHistory) string { return j.Status })

		Expect(len(items)).To(BeNumerically("==", 4))
		Expect(counts[models.StatusSuccess]).To(Equal(2))
		Expect(counts[models.StatusSkipped]).To(Equal(2))
		for _, item := range groups[models.StatusSuccess] {
			Expect(item.TimeEnd).ToNot(BeNil())
			Expect(item.TimeEnd.Sub(item.TimeStart).Milliseconds()).To(BeNumerically("~", 50, 10))
		}
		for _, item := range groups[models.StatusSkipped] {
			Expect(item.TimeEnd).ToNot(BeNil())
			Expect(item.TimeEnd.Sub(item.TimeStart).Milliseconds()).To(BeNumerically("~", 10, 20))
		}
		sampleJob.Singleton = false
		sampleJob.Run()
		sampleJob.Run()
		sampleJob.Run()

		Eventually(func() []models.JobHistory {
			items, _ := sampleJob.FindHistory()
			time.Sleep(time.Millisecond * 250)
			return items
		}, "10s").Should(HaveLen(3))
	})
})
