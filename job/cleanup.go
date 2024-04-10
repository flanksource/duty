package job

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func CleanupStaleHistory(ctx context.Context, age time.Duration, name, resourceID string) error {
	ctx = ctx.WithName(fmt.Sprintf("job=%s", name)).WithName(fmt.Sprintf("resourceID=%s", resourceID))
	query := ctx.DB().Where("NOW() - time_start >= %s", age)
	if name != "" {
		query = query.Where("name = ?", name)
	}
	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}
	res := query.Delete(&models.JobHistory{})
	if res.Error != nil {
		return res.Error
	}

	ctx.Logger.V(1).Infof("Cleaned up %d stale history", res.RowsAffected)
	return nil
}
