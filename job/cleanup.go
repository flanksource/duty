package job

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
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

func CleanupStaleAgentHistory(ctx context.Context, itemsToRetain int) (int, error) {
	query := `
        WITH grouped_history AS (
            SELECT
                id,
                ROW_NUMBER() OVER (
                    PARTITION BY resource_type, resource_id, name, status, agent_id
                    ORDER BY time_start DESC
                ) AS rn
            FROM
                job_history
        )
        DELETE FROM job_history
        WHERE id IN (
            SELECT id
            FROM grouped_history
            WHERE
                rn > ? AND
                agent_id != ?
        )`

	res := ctx.DB().Exec(query, itemsToRetain, uuid.Nil)
	if res.Error != nil {
		return 0, db.ErrorDetails(res.Error)
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
