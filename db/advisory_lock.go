package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrAdvisoryLockTimeout is returned by WithAdvisoryLock when the lock couldn't be taken within its retries.
var ErrAdvisoryLockTimeout = errors.New("timed out waiting for the advisory lock")

// AdvisoryLock is a transaction-scoped Postgres advisory lock, taken exclusively, with a bounded wait.
type AdvisoryLock struct {
	// Key names the lock. It is hashed with hashtext, so every holder must use the same key.
	Key string

	// Timeout is how long each attempt waits for the lock before backing off.
	Timeout time.Duration

	// Retries is how many times to retry after Timeout.
	Retries int

	// Backoff is the wait before the first retry. It doubles after each one.
	Backoff time.Duration
}

// WithAdvisoryLock runs fn holding the lock exclusively, in a savepoint of db's transaction (or a new transaction).
// Each attempt waits for the lock at most lock.Timeout, then backs off and retries, lock.Retries times, so a busy
// lock doesn't stall writers queued behind the exclusive request. The lock is released when the transaction ends.
func WithAdvisoryLock(db *gorm.DB, lock AdvisoryLock, fn func(tx *gorm.DB) error) error {
	backoff := lock.Backoff
	for attempt := 0; ; attempt++ {
		err := db.Transaction(func(tx *gorm.DB) error {
			var previous string
			if err := tx.Raw("SELECT current_setting('lock_timeout')").Scan(&previous).Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT set_config('lock_timeout', ?, true)", fmt.Sprintf("%dms", lock.Timeout.Milliseconds())).Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lock.Key).Error; err != nil {
				return err
			}
			if err := tx.Exec("SELECT set_config('lock_timeout', ?, true)", previous).Error; err != nil {
				return err
			}
			return fn(tx)
		})

		var pgErr *pgconn.PgError
		if err == nil || !errors.As(err, &pgErr) || pgErr.Code != pgerrcode.LockNotAvailable {
			return err
		} else if attempt >= lock.Retries {
			return fmt.Errorf("%w %q", ErrAdvisoryLockTimeout, lock.Key)
		}

		time.Sleep(backoff)
		backoff *= 2
	}
}
