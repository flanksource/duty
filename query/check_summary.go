package query

import (
	gocontext "context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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

	var args []any
	if opt.DeleteFrom != nil {
		query += " OR deleted_at > @from"
		args = append(args, sql.Named("from", *opt.DeleteFrom))
	}
	if opt.Labels != nil {
		query += " AND labels @> @labels"
		args = append(args, sql.Named("labels", opt.Labels))
	}

	rows, err := ctx.DB().Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.CheckSummary
	for rows.Next() {
		var jsonData []byte
		if err := rows.Scan(&jsonData); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if jsonData == nil {
			continue
		}

		var summaries []models.CheckSummary
		if err := json.Unmarshal(jsonData, &summaries); err != nil {
			return nil, api.Errorf(api.EINVALID, "failed to unmarshal check summaries: %v", err)
		}

		results = append(results, summaries...)
	}

	return results, nil
}
