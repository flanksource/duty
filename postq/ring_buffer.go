package postq

import (
	"container/ring"

	"github.com/flanksource/duty/models"
)

func getRecords(ringBuffer *ring.Ring) []models.Event {
	events := make([]models.Event, 0, ringBuffer.Len())
	ringBuffer.Do(func(v any) {
		if v == nil {
			return
		}

		e := v.(models.Event)
		events = append(events, e)
	})

	return events
}
