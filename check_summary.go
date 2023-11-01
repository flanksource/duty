package duty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flanksource/duty/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type CheckSummarySortBy string

var CheckSummarySortByName CheckSummarySortBy = "name"

type CheckSummaryOptions struct {
	SortBy CheckSummarySortBy
}

func OrderByName() CheckSummaryOptions {
	return CheckSummaryOptions{
		SortBy: CheckSummarySortByName,
	}
}

func CheckSummary(ctx DBContext, checkID string) (*models.CheckSummary, error) {
	var checkSummary models.CheckSummary
	if err := ctx.DB().First(&checkSummary, "id = ?", checkID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &checkSummary, nil
}

func QueryCheckSummary(ctx context.Context, dbpool *pgxpool.Pool, opts ...CheckSummaryOptions) (models.Checks, error) {
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

	if len(opts) > 0 && opts[0].SortBy != "" {
		slice := []*models.Check(results)
		slices.SortFunc(slice, func(a, b *models.Check) int {
			var _a, _b string
			if opts[0].SortBy == CheckSummarySortByName {
				_a = a.Name
				_b = b.Name
			}
			if _a > _b {
				return 1
			}
			if _a == _b {
				return 0
			}
			return -1
		})
		return models.Checks(slice), nil
	}

	return results, nil
}

func RefreshCheckStatusSummary(dbpool *pgxpool.Pool) error {
	_, err := dbpool.Exec(context.Background(), `REFRESH MATERIALIZED VIEW check_status_summary`)
	return err
}
