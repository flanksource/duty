package pg

import (
	"strings"

	"github.com/flanksource/duty/context"
)

type routeExtractorFn func(string) (string, string, error)

func defaultRouteExtractor(payload string) (string, string, error) {
	// The original payload is expected to be in the form of
	// <route> <...optional payload>
	fields := strings.Fields(payload)
	route := fields[0]
	derivedPayload := strings.Join(fields[1:], " ")
	return route, derivedPayload, nil
}

// notifyRouter distributes the pgNotify event to multiple channels
// based on the payload.
type notifyRouter struct {
	registry       map[string]chan string
	routeExtractor routeExtractorFn
}

func NewNotifyRouter() *notifyRouter {
	return &notifyRouter{
		registry:       make(map[string]chan string),
		routeExtractor: defaultRouteExtractor,
	}
}

func (t *notifyRouter) WithRouteExtractor(routeExtractor routeExtractorFn) *notifyRouter {
	t.routeExtractor = routeExtractor
	return t
}

// RegisterRoutes creates a single channel for the given routes and returns it.
func (t *notifyRouter) RegisterRoutes(routes ...string) <-chan string {
	// If any of the routes already has a channel, we use that
	// for all the routes.
	// Caution: The caller needs to ensure that the route
	// groups do not overlap.
	pgNotifyChannel := make(chan string)
	for _, we := range routes {
		if existing, ok := t.registry[we]; ok {
			pgNotifyChannel = existing
		}
	}

	for _, we := range routes {
		t.registry[we] = pgNotifyChannel
	}

	return pgNotifyChannel
}

func (t *notifyRouter) Run(ctx context.Context, channel string) {
	eventQueueNotifyChannel := make(chan string)
	go Listen(ctx, channel, eventQueueNotifyChannel)

	for payload := range eventQueueNotifyChannel {
		if payload == "" {
			continue
		}

		route, extractedPayload, err := t.routeExtractor(payload)
		if err != nil {
			continue
		}

		if _, ok := t.registry[route]; !ok {
			continue
		}

		if ch, ok := t.registry[route]; ok {
			go func() {
				ch <- extractedPayload
			}()
		}
	}
}
