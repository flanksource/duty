package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gocontext "context"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

var DefaultQueryTimeout = 30 * time.Second

type CheckSummarySortBy string

var CheckSummarySortByName CheckSummarySortBy = "name"

type CheckSummaryOptions struct {
	Timeout    time.Duration
	CheckID    *uuid.UUID
	SortBy     CheckSummarySortBy
	DeleteFrom *time.Time
}

func OrderByName() CheckSummaryOptions {
	return CheckSummaryOptions{
		SortBy: CheckSummarySortByName,
	}
}

func CheckSummaryByID(ctx context.Context, checkID string) (*models.CheckSummary, error) {
	var checkSummary models.CheckSummary
	if err := ctx.DB().First(&checkSummary, "id = ?", checkID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &checkSummary, nil
}

func CheckSummary(ctx context.Context, opts ...CheckSummaryOptions) (models.Checks, error) {

	opt := CheckSummaryOptions{
		Timeout: DefaultQueryTimeout,
	}
	if len(opts) > 1 {
		return nil, fmt.Errorf("do not specify more than 1 options")
	}
	if len(opts) == 1 {
		opt = opts[0]
	}
	if opt.Timeout.Milliseconds() == 0 {
		opt.Timeout = DefaultQueryTimeout
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(opt.Timeout)
		defer cancel()
	}

	query := `SELECT json_agg(result) FROM check_summary AS result WHERE deleted_at is null`

	var args = pgx.NamedArgs{}
	if opt.DeleteFrom != nil {
		query += " OR deleted_at > @from"
		args["from"] = *opt.DeleteFrom
	}
	rows, err := ctx.Pool().Query(ctx, query, args)
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

func RefreshCheckStatusSummary(ctx context.Context) error {
	_, err := ctx.Pool().Exec(gocontext.Background(), `REFRESH MATERIALIZED VIEW check_status_summary`)
	return err
}
