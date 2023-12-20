package job

import (
	"database/sql"
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gorm.io/gorm/clause"
)

func RefreshCheckStatusSummary(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW check_status_summary").Error
}
func RefreshCheckStatusSummaryAged(ctx context.Context) error {
	return ctx.DB().Exec("REFRESH MATERIALIZED VIEW check_status_summary_aged").Error
}

func DeleteOldCheckStatuses(ctx context.Context, retention int) (error, int) {
	tx := ctx.DB().Exec(`DELETE FROM check_statuses WHERE (NOW() - created_at) > INTERVAL '1 day' * ?`, retention)
	return tx.Error, int(tx.RowsAffected)
}

func DeleteOldCheckStatuses1d(ctx context.Context, retention int) (error, int) {
	tx := ctx.DB().Exec(`DELETE FROM check_statuses_1d WHERE (NOW() - created_at) > INTERVAL '1 day' * ?`, retention)
	return tx.Error, int(tx.RowsAffected)
}

func DeleteOldCheckStatuses1h(ctx context.Context, retention int) (error, int) {
	tx := ctx.DB().Exec(`DELETE FROM check_statuses_1h WHERE (NOW() - created_at) > INTERVAL '1 day' * ?`, retention)
	return tx.Error, int(tx.RowsAffected)
}

func AggregateCheckStatus1d(ctx context.Context) (error, int) {
	const query = `
	SELECT
		check_statuses.check_id,
		date_trunc(?, "time"),
		count(*) AS total_checks,
		count(*) FILTER (WHERE check_statuses.status = TRUE) AS passed,
		count(*) FILTER (WHERE check_statuses.status = FALSE) AS failed,
		SUM(duration) AS duration
	FROM check_statuses
	LEFT JOIN checks ON check_statuses.check_id = checks.id
	WHERE checks.created_at > NOW() - INTERVAL '1 hour' * ?
	GROUP BY 1, 2
	ORDER BY 1,	2 DESC`

	var rows *sql.Rows
	var err error
	rows, err = ctx.DB().Raw(query, "day", 3*24).Rows() // Only look for aggregated data in the last 3 days
	if err != nil {
		return fmt.Errorf("error aggregating check statuses 1h: %w", err), 0
	} else if rows.Err() != nil {
		return fmt.Errorf("error aggregating check statuses 1h: %w", rows.Err()), 0
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var aggr models.CheckStatusAggregate1d
		if err := rows.Scan(&aggr.CheckID, &aggr.CreatedAt, &aggr.Total, &aggr.Passed, &aggr.Failed, &aggr.Duration); err != nil {
			return fmt.Errorf("error scanning aggregated check statuses: %w", err), 0
		}

		cols := []clause.Column{{Name: "check_id"}, {Name: "created_at"}}
		if err := ctx.DB().Clauses(clause.OnConflict{Columns: cols, UpdateAll: true}).Create(aggr).Error; err != nil {
			return fmt.Errorf("error upserting canaries: %w", err), 0
		}
	}
	return nil, count
}

func AggregateCheckStatus1h(ctx context.Context) (error, int) {
	const query = `
	SELECT
		check_statuses.check_id,
		date_trunc(?, "time"),
		count(*) AS total_checks,
		count(*) FILTER (WHERE check_statuses.status = TRUE) AS passed,
		count(*) FILTER (WHERE check_statuses.status = FALSE) AS failed,
		SUM(duration) AS duration
	FROM check_statuses
	LEFT JOIN checks ON check_statuses.check_id = checks.id
	WHERE checks.created_at > NOW() - INTERVAL '1 hour' * ?
	GROUP BY 1, 2
	ORDER BY 1,	2 DESC`

	var rows *sql.Rows
	var err error
	rows, err = ctx.DB().Raw(query, "hour", 3).Rows() // Only look for aggregated data in the last 3 hour
	if err != nil {
		return fmt.Errorf("error aggregating check statuses 1h: %w", err), 0
	} else if rows.Err() != nil {
		return fmt.Errorf("error aggregating check statuses 1h: %w", rows.Err()), 0
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count += 0
		var aggr models.CheckStatusAggregate1h
		if err := rows.Scan(&aggr.CheckID, &aggr.CreatedAt, &aggr.Total, &aggr.Passed, &aggr.Failed, &aggr.Duration); err != nil {
			return fmt.Errorf("error scanning aggregated check statuses: %w", err), 0
		}

		cols := []clause.Column{{Name: "check_id"}, {Name: "created_at"}}
		if err := ctx.DB().Clauses(clause.OnConflict{Columns: cols, UpdateAll: true}).Create(aggr).Error; err != nil {
			return fmt.Errorf("error upserting canaries: %w", err), 0
		}
	}

	return nil, count
}
