package postq

import (
	"container/ring"
	gocontext "context"
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// SyncEventHandlerFunc processes a single event and ONLY makes db changes.
type SyncEventHandlerFunc func(context.Context, models.Event) error

type SyncEventConsumer struct {
	eventLog *ring.Ring

	// Name of the events in the push queue to watch for.
	WatchEvents []string

	// List of sync event handlers that process a single event one after another in order.
	// All the handlers must succeed or else the event will be marked as failed.
	Consumers []SyncEventHandlerFunc

	// ConsumerOption is the configuration for the PGConsumer.
	ConsumerOption *ConsumerOption

	// EventFetcherOption contains configuration on how the events should be fetched.
	EventFetchOption *EventFetcherOption
}

// RecordEvents will record all the events fetched by the consumer in a ring buffer.
func (t *SyncEventConsumer) RecordEvents(size int) {
	t.eventLog = ring.New(size)
}

func (t SyncEventConsumer) GetRecords() ([]models.Event, error) {
	if t.eventLog == nil {
		return nil, fmt.Errorf("event log is not initialized")
	}

	return getRecords(t.eventLog), nil
}

func (t SyncEventConsumer) EventConsumer() (*PGConsumer, error) {
	return NewPGConsumer(t.Handle, t.ConsumerOption)
}

func (t *SyncEventConsumer) Handle(ctx context.Context) (int, error) {
	event, err := t.consumeEvent(ctx)
	if err != nil {
		if event == nil {
			return 0, err
		}

		event.Attempts++
		event.SetError(err.Error())
		const query = `UPDATE event_queue SET error=$1, attempts=$2, last_attempt=NOW() WHERE id=$3`
		if _, err := ctx.Pool().Exec(ctx, query, event.Error, event.Attempts, event.ID); err != nil {
			ctx.Debugf("error saving event attempt updates to event_queue: %v\n", err)
		}
	}

	var eventCount int
	if event != nil {
		eventCount = 1
	}

	return eventCount, err
}

// consumeEvent fetches a single event and passes it to all the consumers in one single transaction.
func (t *SyncEventConsumer) consumeEvent(ctx context.Context) (*models.Event, error) {
	ctx = ctx.WithName("postq").WithDBLogger("postq", logger.Trace)
	tx := ctx.DB().Begin()
	defer tx.Rollback()

	events, err := fetchEvents(ctx, tx, t.WatchEvents, 1, t.EventFetchOption)
	if err != nil {
		return nil, fmt.Errorf("error fetching events: %w", err)
	}

	if len(events) == 0 {
		return nil, nil
	}

	// sync consumers always fetch a single event at a time
	event := events[0]
	if t.eventLog != nil {
		t.eventLog.Value = event
		t.eventLog = t.eventLog.Next()
	}

	for _, syncConsumer := range t.Consumers {
		c := ctx.Wrap(gocontext.Background())
		if err := syncConsumer(c, event); err != nil {
			return &event, err
		}
	}

	return &event, tx.Commit().Error
}

// SyncHandlers converts the given user defined handlers into sync event handlers.
func SyncHandlers(fn ...func(ctx context.Context, e models.Event) error) []SyncEventHandlerFunc {
	var syncHandlers []SyncEventHandlerFunc

	for i := range fn {
		f := fn[i]
		syncHandler := func(ctx context.Context, e models.Event) error {
			return f(ctx, e)
		}
		syncHandlers = append(syncHandlers, syncHandler)
	}

	return syncHandlers
}
