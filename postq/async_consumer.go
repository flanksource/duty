package postq

import (
	"container/ring"
	gocontext "context"
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// AsyncEventHandlerFunc processes multiple events and returns the failed ones
type AsyncEventHandlerFunc func(context.Context, models.Events) models.Events

type AsyncEventConsumer struct {
	eventLog *ring.Ring

	// Name of the events in the push queue to watch for.
	WatchEvents []string

	// Number of events to be fetched and processed at a time.
	BatchSize int

	// An async event handler that consumes events.
	Consumer AsyncEventHandlerFunc

	// ConsumerOption is the configuration for the PGConsumer.
	ConsumerOption *ConsumerOption

	// EventFetcherOption contains configuration on how the events should be fetched.
	EventFetcherOption *EventFetcherOption
}

// RecordEvents will record all the events fetched by the consumer in a ring buffer.
func (t *AsyncEventConsumer) RecordEvents(size int) {
	t.eventLog = ring.New(size)
}

func (t AsyncEventConsumer) GetRecords() ([]models.Event, error) {
	if t.eventLog == nil {
		return nil, fmt.Errorf("event log is not initialized")
	}

	return getRecords(t.eventLog), nil
}

func (t *AsyncEventConsumer) Handle(ctx context.Context) (int, error) {
	ctx = ctx.WithName("postq").WithDBLogger("postq", logger.Trace)
	tx := ctx.DB().Begin()
	defer tx.Rollback() //nolint:errcheck

	events, err := fetchEvents(ctx, tx, t.WatchEvents, t.BatchSize, t.EventFetcherOption)
	if err != nil {
		return 0, fmt.Errorf("error fetching events: %w", err)
	}

	if t.eventLog != nil {
		for _, event := range events {
			t.eventLog.Value = event
			t.eventLog = t.eventLog.Next()
		}
	}
	c := ctx.Wrap(gocontext.Background())
	failedEvents := t.Consumer(c, events)
	if err := failedEvents.Recreate(ctx, tx); err != nil {
		ctx.Debugf("error saving event attempt updates to event_queue: %v\n", err)
	}

	return len(events), tx.Commit().Error
}

func (t AsyncEventConsumer) EventConsumer() (*PGConsumer, error) {
	return NewPGConsumer(t.Handle, t.ConsumerOption)
}

// AsyncHandler converts the given user defined handler into a async event handler.
func AsyncHandler(fn func(ctx context.Context, e models.Events) models.Events) AsyncEventHandlerFunc {
	return func(ctx context.Context, e models.Events) models.Events {
		return fn(ctx, e)
	}
}
