package query

import (
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
)

var syncConfigCacheJob = &job.Job{
	Name:          "SyncConfigCache",
	Schedule:      "@every 5m",
	JobHistory:    true,
	RunNow:        true,
	JitterDisable: true,
	Retention:     job.RetentionFew,
	Fn: func(ctx job.JobRuntime) error {
		return SyncConfigCache(ctx.Context)
	},
}

var updateTypesCache = &job.Job{
	Name:          "UpdateTypesCache",
	Schedule:      "@every 5m",
	Retention:     job.RetentionFailed,
	Singleton:     true,
	JobHistory:    true,
	RunNow:        true,
	JitterDisable: true,
	Fn: func(ctx job.JobRuntime) error {
		return PopulateAllTypesCache(ctx.Context)
	},
}

func PopulateAllTypesCache(ctx context.Context) error {
	var types []string
	query := `SELECT type FROM config_items UNION SELECT type FROM components UNION SELECT type FROM checks`
	if err := ctx.DB().Raw(query).Find(&types).Error; err != nil {
		return err
	}

	allTypesCache.Swap(types)
	return nil
}

var Jobs = []*job.Job{
	syncConfigCacheJob,
	updateTypesCache,
}
