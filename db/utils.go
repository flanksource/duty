package db

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var pgMajorVersion atomic.Int32

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

func IsDBError(err error) bool {
	if oe, ok := oops.AsOops(err); ok {
		if lo.Contains(oe.Tags(), "db") {
			return true
		}
	}

	return false
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

// PGMajorVersion retrieves the PostgreSQL major version
func PGMajorVersion(db *gorm.DB) (int, error) {
	version := pgMajorVersion.Load()
	if version != 0 {
		return int(version), nil
	}

	var versionNum int
	err := db.Raw("SELECT current_setting('server_version_num')::integer;").Scan(&versionNum).Error
	if err != nil {
		return 0, fmt.Errorf("failed to query postgresql version number: %w", err)
	}

	newVersion := int32(versionNum / 10_000)
	pgMajorVersion.Store(newVersion)
	return int(newVersion), nil
}

// ReadTable reads a postgres table when the table model isn't known.
func ReadTable(db *gorm.DB, tableName string, clauses ...clause.Expression) ([]map[string]any, error) {
	rows, err := db.Table(tableName).Clauses(clauses...).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to read table %s: %w", tableName, err)
	}
	defer rows.Close()

	result, err := ScanRows[map[string]any](rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	return result, nil
}
