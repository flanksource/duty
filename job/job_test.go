package job

import (
	"context"
	"testing"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

func TestStatusRing(t *testing.T) {
	var ch = make(chan uuid.UUID, 50)

	cases := []Retention{
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
		{Success: 3, Failed: 3},
	}
	var total int

	eg, _ := errgroup.WithContext(context.TODO())
	eg.Go(func() error {
		for {
			items, _, _, _ := lo.BufferWithTimeout(ch, 32, time.Second*5)
			total += len(items)
			if len(ch) == 0 {
				break
			}
		}
		return nil
	})

	for i := range cases {
		td := cases[i]
		eg.Go(func() error {
			sr := newStatusRing(td, ch)
			for i := 0; i < 100; i++ {
				sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusSuccess)})
				sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusFinished)})

				sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusFailed)})
				sr.Add(&models.JobHistory{ID: uuid.New(), Status: string(models.StatusWarning)})
			}
			return nil
		})
	}

	_ = eg.Wait()
	total += len(ch)

	// we have added 2000 job  history to the status rings
	// based on retention, 5*6*2 jobs remain in the status rings
	// while the rest of them should have been moved to the evicted channel
	expected := 2000 - (5 * 6 * 2)
	if total != expected {
		t.Fatalf("Expected %d job ids in the channel. Got %d", expected, total)
	}
}
