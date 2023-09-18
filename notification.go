package duty

import (
	"fmt"

	"github.com/flanksource/duty/models"
)

// DeleteNotificationSendHistory deletes notification send history
// older than the given duration.
func DeleteNotificationSendHistory(ctx DBContext, days int) (int64, error) {
	tx := ctx.DB().
		Model(&models.NotificationSendHistory{}).
		Where(fmt.Sprintf("created_at < NOW() - INTERVAL '%d DAYS'", days)).
		Delete(&models.NotificationSendHistory{})
	return tx.RowsAffected, tx.Error
}
