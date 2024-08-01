package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func ErrorDetails(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		var errString []string
		if pgErr.Detail != "" {
			errString = append(errString, fmt.Sprintf("detail: %s", pgErr.Detail))
		}
		if pgErr.Hint != "" {
			errString = append(errString, fmt.Sprintf("hint: %s", pgErr.Hint))
		}
		if pgErr.Position != 0 {
			errString = append(errString, fmt.Sprintf(", position: %d", pgErr.Position))
		}
		if len(errString) > 0 {
			return fmt.Errorf("%w: %s", err, strings.Join(errString, ", "))
		}
	}
	return err
}

func IsForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.ForeignKeyViolation
	}

	return false
}

func IsDeadlockError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.DeadlockDetected
	}

	return false
}
