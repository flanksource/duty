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

	// Labels apply to both the canary and check labels
	Labels map[string]string
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

func CheckSummary(ctx context.Context, opts ...CheckSummaryOptions) ([]models.CheckSummary, error) {
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

	selectField := "result"
	switch opt.SortBy {
	case CheckSummarySortByName:
		selectField += " ORDER BY name"
	case "-" + CheckSummarySortByName:
		selectField += " ORDER BY name DESC"
	}

	query := fmt.Sprintf(`SELECT json_agg(%s) FROM check_summary AS result WHERE deleted_at is null`, selectField)

	var args = pgx.NamedArgs{}
	if opt.DeleteFrom != nil {
		query += " OR deleted_at > @from"
		args["from"] = *opt.DeleteFrom
	}
	if opt.Labels != nil {
		query += " AND labels @> @labels"
		args["labels"] = opt.Labels
	}

	rows, err := ctx.Pool().Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.CheckSummary
	for rows.Next() {
		var summaries []models.CheckSummary
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &summaries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components:%v for %s", err, rows.RawValues()[0])
		}

		results = append(results, summaries...)
	}

	return results, nil
}
