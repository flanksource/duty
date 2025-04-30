package query

import (
	"fmt"
	"strings"

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

	ranked := exclause.NewWith(
		"ranked",
		ctx.DB().
			Select(req.baseSelectColumns()).
			Clauses(req.baseWhereClause()...).
			Table("notification_send_history_with_resources"))

	summaryCTE := exclause.NewWith(
		"summary",
		ctx.DB().
			Select(req.summarySelectColumns()).
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
