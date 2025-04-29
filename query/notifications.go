package query

import (
	"fmt"
	"strings"

	extraClausePlugin "github.com/WinterYukky/gorm-extra-clause-plugin"
	"github.com/WinterYukky/gorm-extra-clause-plugin/exclause"
	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// GetNotificationStats retrieves statistics for a notification
func GetNotificationStats(ctx context.Context, notificationIDs ...string) ([]models.NotificationSummary, error) {
	q := ctx.DB()

	if len(notificationIDs) > 0 {
		q = q.Where("id in ?", notificationIDs)
	} else {
		q = q.Where("name != '' AND namespace != '' AND source = ?", models.SourceCRD)
	}

	var summaries []models.NotificationSummary
	if err := q.Find(&summaries).Error; err != nil {
		return nil, fmt.Errorf("error querying notifications_summary: %w", err)
	}

	return summaries, nil
}

func NotificationSendHistorySummary(ctx context.Context, req NotificationSendHistorySummaryRequest) (types.JSON, error) {
	req.SetDefaults()
	if err := req.Validate(); err != nil {
		return nil, api.Errorf(api.EINVALID, "%s", err)
	}

	_ = ctx.DB().Use(extraClausePlugin.New())

	// TODO: Must be dynamic
	selectColumns := []string{
		"resource",
		"resource_type",
		"first_observed",
		"created_at",
		"body",
		"status",
		"ROW_NUMBER() OVER (PARTITION BY resource ORDER BY created_at DESC) AS rn",
	}

	ranked := exclause.NewWith(
		"ranked",
		ctx.DB().
			Select(selectColumns).
			// Where(req.resourceDeletedClause()).
			Clauses(req.baseWhereClause()...).
			Table("notification_send_history_with_resources"))

	// TODO: Must be dynamic
	summaryColumns := []string{
		"resource",
		"resource_type",
		"MIN(first_observed) AS first_observed",
		"MAX(created_at) AS last_seen",
		"COUNT(*) AS total",
		"MAX(CASE WHEN rn = 1 THEN body END) AS last_message",
		"COUNT(CASE WHEN status = 'sent' THEN 1 END) AS sent",
		"COUNT(CASE WHEN status = 'error' THEN 1 END) AS error",
	}

	summaryCTE := exclause.NewWith(
		"summary",
		ctx.DB().
			Select(summaryColumns).
			Table("ranked").
			Group(strings.Join(req.getGroupByColumns(), ",")).
			Order("last_seen DESC"),
	)

	sql := ctx.DB().Clauses(ranked, summaryCTE).Select("json_agg(row_to_json(summary))").Table("summary")

	var res []types.JSON
	if err := sql.Scan(&res).Error; err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0], nil
}
