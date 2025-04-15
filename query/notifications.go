package query

import (
	"fmt"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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
