package job

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

func CleanupStaleHistoryJob(ctx context.Context, age time.Duration, name, resourceID string) *Job {
	return &Job{
		Context:    ctx,
		Name:       "CleanupStaleJobHistory",
		Schedule:   "@every 24h",
		Singleton:  true,
		JobHistory: true,
		Retention:  RetentionFew,
		RunNow:     true,
		Fn: func(ctx JobRuntime) error {
			count, err := cleanupStaleHistory(ctx.Context, age, name, resourceID)
			if err != nil {
				return err
			}

			ctx.History.SuccessCount = count
			return nil
		},
	}
}

func cleanupStaleHistory(ctx context.Context, age time.Duration, name, resourceID string) (int, error) {
	ctx = ctx.WithName(fmt.Sprintf("job=%s", name)).WithName(fmt.Sprintf("resourceID=%s", resourceID))
	query := ctx.DB().Where("NOW() - time_start >= ?", age)
	if name != "" {
		query = query.Where("name = ?", name)
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
