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

// notifyRouter distributes the pgNotify event on a single channel
// to multiple Go channels based on the payload.
type notifyRouter struct {
	// when in signal mode, signals more than the channel size are simply discarded.
	//
	// i.e. if the channel size is 1 & 10 signals come in - we drop the last 8 signals.
	// Essentially, we are squashing the last 9 signals into 1 signal and only publish 2 signals.
	signalMode bool

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

func (t *notifyRouter) GetOrCreateChannel(routes ...string) <-chan string {
	return t.getOrCreateChannel(0, routes...)
}

func (t *notifyRouter) GetOrCreateBufferedChannel(size int, routes ...string) <-chan string {
	t.signalMode = size >= 0
	return t.getOrCreateChannel(size, routes...)
}

// GetOrCreateChannel creates a single channel for the given routes.
//
// If any of the routes already has a channel, we use that existing for all the routes.
//
// Caution: The caller needs to ensure that the route
// groups do not overlap.
func (t *notifyRouter) getOrCreateChannel(size int, routes ...string) <-chan string {
	// we create a channel with size one more than the requested amount
	// so that we have one additional signal in the buffer that represents
	// any further signals that come in.
	pgNotifyChannel := make(chan string, size+1)

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
	go func() {
		err := Listen(ctx, channel, eventQueueNotifyChannel)
		if err != nil {
			ctx.Errorf("notify router listener err: %v", err)
		}
	}()

	t.consume(eventQueueNotifyChannel)
}

func (t *notifyRouter) consume(channel chan string) {
	for payload := range channel {
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
			// publish in a go routine as we don't want any slow consumers
			// to block other fast consumers.
			go func() {
				if t.signalMode {
					select {
					case ch <- extractedPayload:
						// message written
					default:
						// message dropped
					}
				} else {
					ch <- extractedPayload
				}
			}()
		}
	}
}
