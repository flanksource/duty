package duty

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flanksource/duty/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func QueryCheckSummary(ctx context.Context, dbpool *pgxpool.Pool) (models.Checks, error) {
	query := `
    SELECT json_agg(result)
    FROM (
        WITH check_component_relationship_by_check AS (
            SELECT
                check_id,
                json_agg(component_id) AS components
            FROM
                check_component_relationships
            GROUP BY
                check_id
        )
        SELECT
            checks.id::text,
            checks.canary_id::text,
            json_build_object(
                'passed', check_status_summary.passed,
                'failed', check_status_summary.failed,
                'last_pass', check_status_summary.last_pass,
                'last_fail', check_status_summary.last_fail
            ) AS uptime,
            json_build_object('p99', check_status_summary.p99) AS latency,
            checks.last_transition_time,
            checks.type,
            checks.icon,
            checks.name,
            checks.status,
            checks.description,
            canaries.namespace,
            canaries.name as canary_name,
            canaries.labels,
            checks.severity,
            checks.owner,
            checks.last_runtime,
            checks.created_at,
            checks.updated_at,
            checks.deleted_at,
            checks.silenced_at,
            check_component_relationship_by_check.components
        FROM
            checks
            LEFT JOIN check_component_relationship_by_check ON checks.id = check_component_relationship_by_check.check_id
            INNER JOIN canaries ON checks.canary_id = canaries.id
            INNER JOIN check_status_summary ON checks.id = check_status_summary.check_id
    ) AS result
    `
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultQueryTimeout)
		defer cancel()
	}
	rows, err := dbpool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results models.Checks
	for rows.Next() {
		var checks models.Checks
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &checks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components:%v for %s", err, rows.RawValues()[0])
		}
		results = append(results, checks...)
	}

	return results, nil
}

func RefreshCheckStatusSummary(dbpool *pgxpool.Pool) error {
	_, err := dbpool.Exec(context.Background(), `REFRESH MATERIALIZED VIEW check_status_summary`)
	return err
}
