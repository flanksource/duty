package pg

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"

	"github.com/flanksource/duty/context"
)

var (
	DBReconnectMaxDuration         = time.Minute * 5
	DBReconnectBackoffBaseDuration = time.Second
)

// ChannelListener represents a listener for a PostgreSQL notification channel.
type ChannelListener struct {
	Channel  string
	Receiver chan<- string
}

// Listen listens to PostgreSQL notifications on the specified channel.
// It acquires a dedicated connection.
//
// The function blocks until the context is cancelled or an error occurs.
// On connection failure, it will automatically reconnect with exponential backoff.
func Listen(ctx context.Context, channel string, listener chan<- string) error {
	ctx = ctx.WithName("Listen")

	backoff := retry.WithMaxDuration(DBReconnectMaxDuration, retry.NewExponential(DBReconnectBackoffBaseDuration))
	return retry.Do(ctx, backoff, func(retryCtx gocontext.Context) error {
		if err := listenLoop(ctx, map[string][]chan<- string{channel: {listener}}); err != nil {
			ctx.Debugf("listen loop failed, retrying: %v", err)
			return retry.RetryableError(err)
		}
		return nil
	})
}

// ListenMany listens to PostgreSQL notifications on the specified channels.
// It acquires a dedicated connection.
//
// Multiple channels can be listened to and
// multiple listeners can be registered for the same channel - notifications will be sent to all receivers for that channel.
//
// The function blocks until the context is cancelled or an error occurs.
func ListenMany(ctx context.Context, listeners ...ChannelListener) error {
	ctx = ctx.WithName("ListenMany")

	if len(listeners) == 0 {
		return fmt.Errorf("no listeners provided")
	}

	// Group listeners by channel - multiple receivers can listen to same channel
	channels := make(map[string][]chan<- string)
	for _, listener := range listeners {
		channels[listener.Channel] = append(channels[listener.Channel], listener.Receiver)
	}

	backoff := retry.WithMaxDuration(DBReconnectMaxDuration, retry.NewExponential(DBReconnectBackoffBaseDuration))
	return retry.Do(ctx, backoff, func(retryCtx gocontext.Context) error {
		if err := listenLoop(ctx, channels); err != nil {
			ctx.Debugf("listen loop failed, retrying: %v", err)
			return retry.RetryableError(err)
		}
		return nil
	})
}

func listenLoop(ctx context.Context, channels map[string][]chan<- string) error {
	conn, err := ctx.Pool().Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error acquiring database connection: %w", err)
	}
	defer conn.Release()

	for channel := range channels {
		_, err := conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channel))
		if err != nil {
			return fmt.Errorf("error listening to channel %s: %w", channel, err)
		}

		ctx.Debugf("registered channel %s, total listeners: %d", channel, len(channels[channel]))
	}

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("error waiting for notification: %w", err)
		}

		recipients := channels[notification.Channel]
		for _, ch := range recipients {
			ch <- notification.Payload
		}
	}
}
