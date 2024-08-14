package pg

import (
	"strings"

	"github.com/flanksource/duty/context"
)

// notifyRouter distributes the pgNotify event to multiple channels
// based on the payload.
type notifyRouter struct {
	registry map[string]chan string
}

func NewNotifyRouter() *notifyRouter {
	return &notifyRouter{
		registry: make(map[string]chan string),
	}
}

// RegisterRoutes creates a single channel for the given routes and returns it.
func (t *notifyRouter) RegisterRoutes(routes ...string) <-chan string {
	pgNotifyChannel := make(chan string)
	for _, we := range routes {
		t.registry[we] = pgNotifyChannel
	}

	return pgNotifyChannel
}

func (t *notifyRouter) Run(ctx context.Context, channel string) {
	eventQueueNotifyChannel := make(chan string)
	go Listen(ctx, channel, eventQueueNotifyChannel)

	for payload := range eventQueueNotifyChannel {
		if _, ok := t.registry[payload]; !ok || payload == "" {
			continue
		}

		// The original payload is expected to be in the form of
		// <route> <...optional payload>
		fields := strings.Fields(payload)
		route := fields[0]
		derivedPayload := strings.Join(fields[1:], " ")

		if ch, ok := t.registry[route]; ok {
			ch <- derivedPayload
		}
	}
}
