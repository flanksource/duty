package postq

import (
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/lib/pq"
	"github.com/samber/oops"
	"gorm.io/gorm"
)

type EventFetcherOption struct {
	// MaxAttempts is the number of times an event is attempted to process
	// default: 3
	MaxAttempts int

	// BaseDelay is the base delay between retries
	// default: 60 seconds
	BaseDelay int

	// Exponent is the exponent of the base delay
	// default: 5 (along with baseDelay = 60, the retries are 1, 6, 31, 156 (in minutes))
	Exponent int
}

// fetchEvents fetches given watch events from the `event_queue` table.
func fetchEvents(ctx context.Context, tx *gorm.DB, watchEvents []string, batchSize int, opts *EventFetcherOption) ([]models.Event, error) {
	if batchSize == 0 {
		batchSize = 1
	}

	const selectEventsQuery = `
		WITH to_delete AS (
			SELECT id FROM event_queue
			WHERE
				(delay IS NULL OR created_at + (delay * INTERVAL '1 second' / 1000000000)  <= NOW()) AND
				attempts <= @MaxAttempts AND
				name = ANY(@Events) AND
				(last_attempt IS NULL OR last_attempt <= NOW() - INTERVAL '1 SECOND' * @BaseDelay * POWER(attempts, @Exponent))
			ORDER BY priority DESC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT @BatchSize
		)
		DELETE FROM event_queue
		WHERE id IN (SELECT id FROM to_delete)
		RETURNING *
	`

	type EventArgs struct {
		Events      pq.StringArray
		BatchSize   int
		MaxAttempts int
		BaseDelay   int
		Exponent    int
	}

	args := EventArgs{
		Events:      watchEvents,
		BatchSize:   batchSize,
		MaxAttempts: 3,
		BaseDelay:   60,
		Exponent:    5,
	}

	if opts != nil {
		if opts.MaxAttempts > 0 {
			args.MaxAttempts = opts.MaxAttempts
		}

		if opts.BaseDelay > 0 {
			args.BaseDelay = opts.BaseDelay
		}

		if opts.Exponent > 0 {
			args.Exponent = opts.Exponent
		}
	}
	var events []models.Event

	if err := tx.Raw(selectEventsQuery, args).Scan(&events).Error; err != nil {
		return nil, oops.Tags("db").Wrap(err)
	}

	if len(events) > 0 {
		ctx.Tracef("queue=%s fetched=%d", strings.Join(watchEvents, ","), len(events))
	}

	if len(events) > batchSize {
		ctx.Errorf("fetched more events (%d) than the requested amoun (%d)", len(events), batchSize)
	}

	return events, nil
}
