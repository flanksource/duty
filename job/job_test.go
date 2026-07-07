package job

import (
	"context"
	"testing"
	"time"

	"github.com/flanksource/commons/properties"
	dutycontext "github.com/flanksource/duty/context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

func TestJob(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Job Suite")
}

var _ = Describe("Job", func() {
	It("builds IDs from logical job identity", func() {
		cases := []struct {
			name string
			job  *Job
			want string
		}{
			{
				name: "resource identity",
				job: &Job{
					Name:         "Scraper",
					ResourceType: ResourceTypeScraper,
					ResourceID:   "scraper-1",
					Aliases:      []string{"default/scraper"},
				},
				want: "Scraper::config_scraper/scraper-1",
			},
			{
				name: "aliases are not identity",
				job: &Job{
					Name:    "Scraper",
					Aliases: []string{"manual", "scheduled"},
				},
				want: "Scraper",
			},
			{
				name: "name fallback",
				job: &Job{
					Name: "Scraper",
				},
				want: "Scraper",
			},
		}

		for _, tt := range cases {
			Expect(tt.job.ID()).To(Equal(tt.want), tt.name)
		}
	})

	It("keeps property lookup backward compatible", func() {
		props := map[string]string{
			"jobs.compat-job.schedule":                       "general",
			"jobs.compat-job.alias.schedule":                 "specific",
			"jobs.compat-job.retention.success":              "10",
			"jobs.compat-job.alias.retention.success":        "20",
			"jobs.compat-alias-only.alias.schedule":          "specific",
			"jobs.compat-alias-only.alias.retention.success": "20",
			"jobs.schedule":                                  "global",
			"jobs.retention.success":                         "30",
		}
		for key, value := range props {
			properties.Global.Set(key, value)
		}
		DeferCleanup(func() {
			for key := range props {
				properties.Global.Set(key, "")
			}
		})

		j := &Job{
			Name:    "compat-job",
			Aliases: []string{"alias"},
			Context: dutycontext.NewContext(context.Background()),
		}
		schedule, ok := j.GetProperty("schedule")
		Expect(ok).To(BeTrue())
		Expect(schedule).To(Equal("general"))
		Expect(j.GetPropertyInt("retention.success", 1)).To(Equal(10))

		aliasOnly := &Job{
			Name:    "compat-alias-only",
			Aliases: []string{"alias"},
			Context: dutycontext.NewContext(context.Background()),
		}
		schedule, ok = aliasOnly.GetProperty("schedule")
		Expect(ok).To(BeTrue())
		Expect(schedule).To(Equal("specific"))
		Expect(aliasOnly.GetPropertyInt("retention.success", 1)).To(Equal(20))

		globalOnly := &Job{
			Name:    "compat-global-only",
			Aliases: []string{"alias"},
			Context: dutycontext.NewContext(context.Background()),
		}
		schedule, ok = globalOnly.GetProperty("schedule")
		Expect(ok).To(BeFalse())
		Expect(schedule).To(BeEmpty())
		Expect(globalOnly.GetPropertyInt("retention.success", 1)).To(Equal(1))
	})
})

var _ = Describe("StatusRing", Label("slow"), func() {
	var ch chan uuid.UUID

	cases := []Retention{
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
	}
	var total int
	var loops int
	var expected int

	BeforeEach(func() {
		ch = make(chan uuid.UUID, 50)
		total = 0
		loops = 100
		expected = (len(cases) * loops * 3) - (3 * 3 * len(cases))
	})

	It("should process job histories correctly", func() {
		eg, _ := errgroup.WithContext(context.TODO())
		eg.Go(func() error {
			for {
				items, _, _, _ := lo.BufferWithTimeout(ch, 32, time.Second*5)
				total += len(items)
				if total >= expected {
					break
				}
			}
			return nil
		})

		for i := range cases {
			td := cases[i]
			eg.Go(func() error {
				sr := NewStatusRing(td, false, ch)
				for i := 0; i < loops; i++ {
					sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusSuccess)})
					sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusFailed)})
					sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusWarning)})
				}
				return nil
			})
		}

		_ = eg.Wait()
		total += len(ch)

		// we have added 1500 job  history to the status rings
		// based on retention, 5*3*3 (cases * uniq status * retention for uniq status) jobs remain in the status rings
		// while the rest of them should have been moved to the evicted channel
		Expect(total).To(Equal(expected))
	})
})
