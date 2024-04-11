package job

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func CleanupStaleHistoryJob(age time.Duration, name, resourceID string) *Job {
	return &Job{
		Name:       "CleanupStaleJobHistory",
		Schedule:   "@every 24h",
		Singleton:  true,
		JobHistory: true,
		Retention:  RetentionFew,
		RunNow:     true,
		Fn: func(ctx JobRuntime) error {
			return cleanupStaleHistory(ctx.Context, age, name, resourceID)
		},
	}
}

func cleanupStaleHistory(ctx context.Context, age time.Duration, name, resourceID string) error {
	ctx = ctx.WithName(fmt.Sprintf("job=%s", name)).WithName(fmt.Sprintf("resourceID=%s", resourceID))
	query := ctx.DB().Debug().Where("NOW() - time_start >= ?", age)
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
