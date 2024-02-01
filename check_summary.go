package duty

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/jackc/pgx/v5/pgxpool"
)

// deprecated use query.CheckSummaryByID
func CheckSummary(ctx context.Context, checkID string) (*models.CheckSummary, error) {
	return query.CheckSummaryByID(ctx, checkID)
}

// deprecated use query.CheckSummary
func QueryCheckSummary(ctx context.Context, dbpool *pgxpool.Pool, opts ...query.CheckSummaryOptions) ([]models.CheckSummary, error) {
	return query.CheckSummary(ctx, opts...)
}
