package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm"

	"gorm.io/gorm/clause"
)

var DateMapper = func(ctx context.Context, val string) (any, error) {
	if expr, err := datemath.Parse(val); err != nil {
		return nil, fmt.Errorf("invalid date '%s': %s", val, err)
	} else {
		return expr.Time(), nil
	}

}

var AgentMapper = func(ctx context.Context, id string) (any, error) {
	if id, err := uuid.Parse(id); err == nil {
		return id.String(), nil
	}

	if agent, _ := FindCachedAgent(ctx, id); agent != nil && agent.ID != uuid.Nil {
		return &agent.ID, nil
	}
	return nil, fmt.Errorf("invalid agent: %s", id)
}

type QueryModel struct {
	Table        string
	LabelsColumn string
	JSONColumn   string
	DateFields   []string
	Columns      []string
	FieldMapper  map[string]func(ctx context.Context, id string) (any, error)
	Custom       map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error)
	Aliases      map[string]string
}

var ConfigQueryModel = QueryModel{
	Table:      "configs",
	JSONColumn: "config",
	Custom: map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error){
		"limit": func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error) {
			if i, err := strconv.Atoi(val); err == nil {
				return tx.Limit(i), nil
			} else {
				return nil, err
			}
		},
		"sort": func(ctx context.Context, tx *gorm.DB, sort string) (*gorm.DB, error) {
			return tx.Order(clause.OrderByColumn{Column: clause.Column{Name: sort}}), nil
		},
		"offset": func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error) {
			if i, err := strconv.Atoi(val); err == nil {
				return tx.Offset(i), nil
			} else {
				return nil, err
			}
		},
	},
	Columns: []string{
		"name", "source", "type", "status", "health",
	},
	LabelsColumn: "labels",
	Aliases: map[string]string{
		"created":     "created_at",
		"updated":     "updated_at",
		"deleted":     "deleted_at",
		"scraped":     "last_scraped_time",
		"agent":       "agent_id",
		"config_type": "type",
		"namespace":   "@namespace",
	},

	FieldMapper: map[string]func(ctx context.Context, id string) (any, error){
		"agent_id":          AgentMapper,
		"created_at":        DateMapper,
		"updated_at":        DateMapper,
		"deleted_at":        DateMapper,
		"last_scraped_time": DateMapper,
	},
}

var ComponentQueryModel = QueryModel{
	Table:      "components",
	JSONColumn: "component",
	Custom: map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error){
		"limit": func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error) {
			if i, err := strconv.Atoi(val); err == nil {
				return tx.Limit(i), nil
			} else {
				return nil, err
			}
		},
		"sort": func(ctx context.Context, tx *gorm.DB, sort string) (*gorm.DB, error) {
			return tx.Order(clause.OrderByColumn{Column: clause.Column{Name: sort}}), nil
		},
		"offset": func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error) {
			if i, err := strconv.Atoi(val); err == nil {
				return tx.Offset(i), nil
			} else {
				return nil, err
			}
		},
		"component_config_traverse": func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error) {
			// search: component_config_traverse=72143d48-da4a-477f-bac1-1e9decf188a6,outgoing
			// Args should be componentID, direction and types (compID,direction)
			args := strings.Split(val, ",")
			componentID := args[0]
			direction := "outgoing"
			if len(args) > 1 {
				direction = args[1]
			}
			// NOTE: Direction is not supported as of now
			_ = direction
			tx = tx.Where("id IN (SELECT id from lookup_component_config_id_related_components(?))", componentID)
			return tx, nil
		},
	},
	Columns: []string{
		"name", "topology_id", "type", "status", "health",
	},
	LabelsColumn: "labels",
	Aliases: map[string]string{
		"created":        "created_at",
		"updated":        "updated_at",
		"deleted":        "deleted_at",
		"scraped":        "last_scraped_time",
		"agent":          "agent_id",
		"component_type": "type",
		"namespace":      "@namespace",
	},

	FieldMapper: map[string]func(ctx context.Context, id string) (any, error){
		"agent_id":          AgentMapper,
		"created_at":        DateMapper,
		"updated_at":        DateMapper,
		"deleted_at":        DateMapper,
		"last_scraped_time": DateMapper,
	},
}

func GetModelFromTable(table string) (QueryModel, error) {
	switch table {
	case models.ConfigItem{}.TableName():
		return ConfigQueryModel, nil
	case models.Component{}.TableName():
		logger.Infof("Inside comp get")
		return ComponentQueryModel, nil
	default:
		return QueryModel{}, fmt.Errorf("invalid table")
	}
}

func (qm QueryModel) Apply(ctx context.Context, q types.QueryField, tx *gorm.DB) (*gorm.DB, []clause.Expression, error) {
	if tx == nil {
		tx = ctx.DB().Table(qm.Table)
	}
	clauses := []clause.Expression{}
	var err error
	if q.Field != "" {
		q.Field = strings.ToLower(q.Field)
		if alias, ok := qm.Aliases[q.Field]; ok {
			q.Field = alias
		}

		val := fmt.Sprint(q.Value)
		if mapper, ok := qm.FieldMapper[q.Field]; ok {
			if q.Value, err = mapper(ctx, val); err != nil {
				return nil, nil, err
			}

		}

		if mapper, ok := qm.Custom[q.Field]; ok {
			tx, err = mapper(ctx, tx, val)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Invalid value for %s", q.Field)
			}
		}

		if c, err := q.ToClauses(); err != nil {
			return nil, nil, err
		} else {
			clauses = append(clauses, c...)
		}
	}

	for _, f := range q.Fields {
		_tx, _clauses, err := qm.Apply(ctx, *f, tx)
		if err != nil {
			return nil, nil, err
		}
		tx = _tx
		clauses = append(clauses, _clauses...)
	}

	return tx, clauses, nil
}
