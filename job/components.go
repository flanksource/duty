package job

import (
	"time"

	"github.com/flanksource/duty/context"
)

func CleanupSoftDeletedComponents(ctx context.Context, olderThan time.Duration) (int, error) {
	tx := ctx.DB().
		Exec("DELETE FROM components WHERE deleted_at < NOW() - interval '1 SECONDS' * ?", int64(olderThan.Seconds()))
	if tx.Error != nil {
		return 0, tx.Error
	}

	return int(tx.RowsAffected), nil
}
