package job

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func CleanupStaleHistory(ctx context.Context, age time.Duration, name, resourceID string, statuses ...string) (int, error) {
	ctx = ctx.WithName(fmt.Sprintf("job=%s", name)).WithName(fmt.Sprintf("resourceID=%s", resourceID))

	query := ctx.DB().Where("NOW() - time_start >= ?", age)
	if name != "" {
		query = query.Where("name = ?", name)
	}
	if len(statuses) != 0 {
		query = query.Where("status IN ?", statuses)
	}
	if resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	}
	res := query.Delete(&models.JobHistory{})
	if res.Error != nil {
		return 0, res.Error
	}

	return int(res.RowsAffected), nil
}

func CleanupStaleRunningHistory(ctx context.Context, age time.Duration) (int, error) {
	res := ctx.DB().
		Model(&models.JobHistory{}).
		Where("NOW() - time_start >= ?", age).
		Where("status = ?", models.StatusRunning).
		Update("status", models.StatusStale)
	if res.Error != nil {
		return 0, res.Error
	}

	return int(res.RowsAffected), nil
}
