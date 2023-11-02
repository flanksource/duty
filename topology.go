package duty

import (
	gocontext "context"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"

	"github.com/flanksource/duty/query"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type TopologyOptions = query.TopologyOptions
type TopologyResponse = query.TopologyResponse

// deprecated use query.Topology
func QueryTopology(ctx gocontext.Context, dbpool *pgxpool.Pool, params TopologyOptions) (*TopologyResponse, error) {
	return query.Topology(context.NewContext(ctx).WithDB(nil, pool), params)
}

// deprecated use query.GetComponent
func GetComponent(ctx gocontext.Context, db *gorm.DB, id string) (*models.Component, error) {
	return query.GetComponent(context.NewContext(ctx).WithDB(db, nil), id)
}
