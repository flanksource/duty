package config

import (
	gocontext "context"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"

	"github.com/jackc/pgx/v5/pgxpool"
)

// deprecrated Use query.Config
func Query(ctx gocontext.Context, conn *pgxpool.Pool, sqlQuery string) ([]map[string]any, error) {
	return query.Config(context.NewContext(ctx).WithDB(nil, conn), sqlQuery)
}
