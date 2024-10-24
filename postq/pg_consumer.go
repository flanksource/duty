package postq

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
)

type ConsumerFunc func(ctx context.Context) (count int, err error)

// PGConsumer manages concurrent consumers to handle PostgreSQL NOTIFY events from a specific channel.
type PGConsumer struct {
	// Number of concurrent consumers
	numConsumers int

	// pgNotifyTimeout is the timeout to consume events in case no Consume notification is received.
	pgNotifyTimeout time.Duration

	// consumerFunc is responsible in fetching & consuming the events for the given batch size and events.
	// It returns the number of events it fetched.
	consumerFunc ConsumerFunc

	// handle errors when consuming.
	errorHandler func(ctx context.Context, e error) bool
}

type ConsumerOption struct {
	// Number of concurrent consumers.
	// 	default: 1
	NumConsumers int

	// Timeout is the timeout to call the consumer func in case no pg notification is received.
	// 	default: 1 minute
	Timeout time.Duration

	// handle errors when consuming.
	// returns whether to retry or not.
	// 	default: sleep for 1s and retry.
	ErrorHandler func(ctx context.Context, e error) bool
}

// NewPGConsumer returns a new EventConsumer
func NewPGConsumer(consumerFunc ConsumerFunc, opt *ConsumerOption) (*PGConsumer, error) {
	if consumerFunc == nil {
		return nil, fmt.Errorf("consumer func cannot be nil")
	}

	ec := &PGConsumer{
		numConsumers:    1,
		consumerFunc:    consumerFunc,
		pgNotifyTimeout: time.Minute,
		errorHandler:    defaultErrorHandler,
	}

	if opt != nil {
		if opt.Timeout != 0 {
			ec.pgNotifyTimeout = opt.Timeout
		}

		if opt.NumConsumers > 0 {
			ec.numConsumers = opt.NumConsumers
		}

		if opt.ErrorHandler != nil {
			ec.errorHandler = opt.ErrorHandler
		}
	}

	return ec, nil
}

// ConsumeUntilEmpty consumes events in a loop until the event queue is empty.
func (t *PGConsumer) ConsumeUntilEmpty(ctx context.Context) {
	for {
		count, err := t.consumerFunc(ctx)
		if err != nil {
			if !t.errorHandler(ctx, err) {
				return
			}
		} else if count == 0 {
			return
		}
	}
}

// Listen starts consumers in the background
func (e *PGConsumer) Listen(ctx context.Context, pgNotify <-chan string) {
	for i := 0; i < e.numConsumers; i++ {
		go func() {
			for {
				select {
				case <-pgNotify:
					e.ConsumeUntilEmpty(ctx)

				case <-time.After(e.pgNotifyTimeout):
					e.ConsumeUntilEmpty(ctx)
				}
			}
		}()
	}
}

func defaultErrorHandler(ctx context.Context, e error) bool {
	time.Sleep(time.Second)
	ctx.Debugf("default error: %v", e)
	return true
}
