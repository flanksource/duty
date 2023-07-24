package duty

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flanksource/duty/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func QueryCheckSummary(ctx context.Context, dbpool *pgxpool.Pool) (models.Checks, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultQueryTimeout)
		defer cancel()
	}

	query := `SELECT json_agg(result) FROM check_summary AS result`
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
