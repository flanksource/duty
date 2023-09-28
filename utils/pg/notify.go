package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty"
	"github.com/sethvargo/go-retry"
)

// Defaults ...
const (
	dbReconnectMaxDuration         = time.Minute * 5
	dbReconnectBackoffBaseDuration = time.Second
)

// Listen listens to postgres notifications.
// On failure, it'll keep retrying with backoff
func Listen(ctx duty.DBContext, channel string, pgNotify chan<- string) {
	var listen = func(ctx duty.DBContext, pgNotify chan<- string) error {
		conn, err := ctx.Pool().Acquire(ctx)
		if err != nil {
			return fmt.Errorf("error acquiring database connection: %v", err)
		}
		defer conn.Release()

		_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channel))
		if err != nil {
			return fmt.Errorf("error listening to database notifications: %v", err)
		}
		logger.Debugf("listening to database notifications: %s", channel)

		for {
			n, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				return fmt.Errorf("error listening to database notifications: %v", err)
			}

			pgNotify <- n.Payload
		}
	}

	// retry on failure.
	for {
		backoff := retry.WithMaxDuration(dbReconnectMaxDuration, retry.NewExponential(dbReconnectBackoffBaseDuration))
		err := retry.Do(ctx, backoff, func(retryContext context.Context) error {
			ctx := retryContext.(duty.DBContext)
			if err := listen(ctx, pgNotify); err != nil {
				return retry.RetryableError(err)
			}

			return nil
		})

		logger.Errorf("failed to connect to database: %v", err)
	}
}
