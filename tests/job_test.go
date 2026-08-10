package tests

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
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
		_ = context.UpdateProperty(ctx, "job.eviction.period", "1s")
		_ = context.UpdateProperty(ctx, "job.jitter.disable", "true")

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

	It("Prevents concurrent execution for separate job instances with the same resource identity", func() {
		var firstRuns atomic.Int32
		var secondRuns atomic.Int32
		resourceID := uuid.NewString()
		name := "keyed-singleton-" + uuid.NewString()
		started := make(chan struct{})
		release := make(chan struct{})
		firstDone := make(chan struct{})
		var releaseOnce sync.Once

		DeferCleanup(func() {
			releaseOnce.Do(func() { close(release) })
			Eventually(firstDone, "5s").Should(BeClosed())
		})

		first := &job.Job{
			Name:                 name,
			Aliases:              []string{"scheduled"},
			Singleton:            true,
			Context:              DefaultContext,
			ResourceID:           resourceID,
			ResourceType:         job.ResourceTypeScraper,
			IgnoreSuccessHistory: true,
			Fn: func(ctx job.JobRuntime) error {
				firstRuns.Add(1)
				close(started)
				<-release
				return nil
			},
		}
		second := &job.Job{
			Name:                 name,
			Aliases:              []string{"manual"},
			Singleton:            true,
			Context:              DefaultContext,
			ResourceID:           resourceID,
			ResourceType:         job.ResourceTypeScraper,
			IgnoreSuccessHistory: true,
			Fn: func(ctx job.JobRuntime) error {
				secondRuns.Add(1)
				return nil
			},
		}

		go func() {
			defer close(firstDone)
			first.Run()
		}()

		Eventually(started, "5s").Should(BeClosed())
		second.Run()

		Expect(firstRuns.Load()).To(Equal(int32(1)))
		Expect(secondRuns.Load()).To(BeZero())
		Expect(second.LastJob).ToNot(BeNil())
		Expect(second.LastJob.Status).To(Equal(models.StatusSkipped))

		releaseOnce.Do(func() { close(release) })
		Eventually(firstDone, "5s").Should(BeClosed())
	})

	It("Persists the resource identity on skipped job histories", func() {
		resourceID := uuid.NewString()
		name := "skipped-resource-id-" + uuid.NewString()
		started := make(chan struct{})
		release := make(chan struct{})
		firstDone := make(chan struct{})
		var releaseOnce sync.Once
		var runs atomic.Int32

		DeferCleanup(func() {
			releaseOnce.Do(func() { close(release) })
			Eventually(firstDone, "5s").Should(BeClosed())
		})

		makeJob := func() *job.Job {
			return &job.Job{
				Name:         name,
				Singleton:    true,
				JobHistory:   true,
				Context:      DefaultContext,
				ResourceID:   resourceID,
				ResourceType: job.ResourceTypeScraper,
				Fn: func(ctx job.JobRuntime) error {
					if runs.Add(1) == 1 {
						close(started)
						<-release
					}
					return nil
				},
			}
		}

		first := makeJob()
		second := makeJob()

		go func() {
			defer close(firstDone)
			first.Run()
		}()

		Eventually(started, "5s").Should(BeClosed())
		second.Run()

		Expect(second.LastJob).ToNot(BeNil())
		Expect(second.LastJob.Status).To(Equal(models.StatusSkipped))
		Expect(second.LastJob.ResourceID).To(Equal(resourceID))
		Expect(second.LastJob.ResourceType).To(Equal(job.ResourceTypeScraper))

		releaseOnce.Do(func() { close(release) })
		Eventually(firstDone, "5s").Should(BeClosed())

		skipped, err := second.FindHistory(models.StatusSkipped)
		Expect(err).ToNot(HaveOccurred())
		Expect(skipped).ToNot(BeEmpty())
		for _, h := range skipped {
			Expect(h.ResourceID).To(Equal(resourceID))
			Expect(h.ResourceType).To(Equal(job.ResourceTypeScraper))
		}
	})

	It("Retains the resource identity when a running job history goes stale", func() {
		resourceID := uuid.NewString()
		history := models.JobHistory{
			Name:         "stale-resource-id-" + uuid.NewString(),
			ResourceID:   resourceID,
			ResourceType: job.ResourceTypeScraper,
			Status:       models.StatusRunning,
			TimeStart:    time.Now().Add(-time.Hour),
		}
		Expect(DefaultContext.DB().Create(&history).Error).To(BeNil())

		count, err := job.CleanupStaleRunningHistory(DefaultContext, time.Minute)
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(BeNumerically(">=", 1))

		var stale models.JobHistory
		Expect(DefaultContext.DB().Where("id = ?", history.ID).First(&stale).Error).To(BeNil())
		Expect(stale.Status).To(Equal(models.StatusStale))
		Expect(stale.ResourceID).To(Equal(resourceID))
		Expect(stale.ResourceType).To(Equal(job.ResourceTypeScraper))
	})

	It("Allows singleton jobs with different resource identities to run concurrently", func() {
		var firstRuns atomic.Int32
		var secondRuns atomic.Int32
		name := "keyed-singleton-concurrent-" + uuid.NewString()
		firstID := uuid.NewString()
		secondID := uuid.NewString()
		started := make(chan string, 2)
		release := make(chan struct{})
		var wg sync.WaitGroup
		var releaseOnce sync.Once
		waitForJobs := func() {
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			Eventually(done, "5s").Should(BeClosed())
		}

		DeferCleanup(func() {
			releaseOnce.Do(func() { close(release) })
			waitForJobs()
		})

		makeJob := func(resourceID string, runs *atomic.Int32) *job.Job {
			return &job.Job{
				Name:                 name,
				Singleton:            true,
				Context:              DefaultContext,
				ResourceID:           resourceID,
				ResourceType:         job.ResourceTypeScraper,
				IgnoreSuccessHistory: true,
				Fn: func(ctx job.JobRuntime) error {
					runs.Add(1)
					started <- resourceID
					<-release
					return nil
				},
			}
		}

		runAsync := func(j *job.Job) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				j.Run()
			}()
		}

		first := makeJob(firstID, &firstRuns)
		second := makeJob(secondID, &secondRuns)

		runAsync(first)
		Eventually(started, "5s").Should(Receive(Equal(firstID)))
		runAsync(second)
		Eventually(started, "5s").Should(Receive(Equal(secondID)))

		Expect(firstRuns.Load()).To(Equal(int32(1)))
		Expect(secondRuns.Load()).To(Equal(int32(1)))

		releaseOnce.Do(func() { close(release) })
		waitForJobs()
		Expect(first.LastJob.Status).To(Equal(models.StatusSuccess))
		Expect(second.LastJob.Status).To(Equal(models.StatusSuccess))
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

	PIt("Should clean up jobs", func() {
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
