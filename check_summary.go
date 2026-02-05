package duty

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
)

// deprecated use query.CheckSummaryByID
func CheckSummary(ctx context.Context, checkID string) (*models.CheckSummary, error) {
	return query.CheckSummaryByID(ctx, checkID)
}
