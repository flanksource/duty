package job

import (
	"context"
	"testing"
	"time"

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
