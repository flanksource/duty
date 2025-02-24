package query

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query/grammar"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/samber/lo"
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

var JSONPathMapper = func(ctx context.Context, tx *gorm.DB, column string, op grammar.QueryOperator, path string, val string) *gorm.DB {
	if !slices.Contains([]grammar.QueryOperator{grammar.Eq, grammar.Neq}, op) {
		op = grammar.Eq
	}
	values := strings.Split(val, ",")
	for _, v := range values {
		tx = tx.Where(fmt.Sprintf(`TRIM(BOTH '"' from jsonb_path_query_first(%s, '$.%s')::TEXT) %s ?`, column, path, op), v)
	}
	return tx
}

var CommonFields = map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error){
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
}

type QueryModel struct {
	Table  string
	Custom map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error)

	// List of jsonb columns that store a map.
	// These columns can be addressed using dot notation to access the JSON fields directly
	// Example: tags.cluster or tags.namespace.
	JSONMapColumns []string

	// List of columns that can be addressed on the search query.
	// Any other fields will be treated as a property lookup.
	Columns []string

	// Alias maps fields from the search query to the table columns
	Aliases map[string]string

	// True when the table has a "tags" column
	HasTags bool

	// True when the table has a "labels" column
	HasLabels bool

	// True when the table has an "agent_id" column
	HasAgents bool

	// True when the table has properties column
	HasProperties bool

	// FieldMapper maps the value of these fields
	FieldMapper map[string]func(ctx context.Context, id string) (any, error)
}

var ConfigQueryModel = QueryModel{
	Table: models.ConfigItem{}.TableName(),
	Columns: []string{
		"name", "source", "type", "status", "agent_id", "health", "external_id", "config_class",
		"created_at", "updated_at", "deleted_at", "last_scraped_time",
	},
	JSONMapColumns: []string{"labels", "tags", "config"},
	HasProperties:  true,
	HasTags:        true,
	HasAgents:      true,
	HasLabels:      true,
	Aliases: map[string]string{
		"created":     "created_at",
		"updated":     "updated_at",
		"deleted":     "deleted_at",
		"scraped":     "last_scraped_time",
		"agent":       "agent_id",
		"config_type": "type",
		"namespace":   "tags.namespace",
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
	Table: models.Component{}.TableName(),
	Custom: map[string]func(ctx context.Context, tx *gorm.DB, val string) (*gorm.DB, error){
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
		"name", "namespace", "topology_id", "type", "status", "health", "agent_id",
		"created_at", "updated_at", "deleted_at",
	},
	JSONMapColumns: []string{"labels", "summary"},
	Aliases: map[string]string{
		"created":        "created_at",
		"updated":        "updated_at",
		"deleted":        "deleted_at",
		"agent":          "agent_id",
		"component_type": "type",
	},
	HasProperties: true,
	HasAgents:     true,
	HasLabels:     true,
	FieldMapper: map[string]func(ctx context.Context, id string) (any, error){
		"agent_id":          AgentMapper,
		"created_at":        DateMapper,
		"updated_at":        DateMapper,
		"deleted_at":        DateMapper,
		"last_scraped_time": DateMapper,
	},
}

var CheckQueryModel = QueryModel{
	Table: models.Check{}.TableName(),
	Columns: []string{
		"name", "namespace", "canary_id", "type", "status", "agent_id",
		"created_at", "updated_at", "deleted_at",
	},
	JSONMapColumns: []string{"spec", "labels"},
	Aliases: map[string]string{
		"created":    "created_at",
		"updated":    "updated_at",
		"deleted":    "deleted_at",
		"agent":      "agent_id",
		"health":     "status",
		"check_type": "type",
	},
	HasAgents: true,
	HasLabels: true,
	FieldMapper: map[string]func(ctx context.Context, id string) (any, error){
		"agent_id":   AgentMapper,
		"created_at": DateMapper,
		"updated_at": DateMapper,
		"deleted_at": DateMapper,
	},
}

var PlaybookQueryModel = QueryModel{
	Table:   models.Playbook{}.TableName(),
	HasTags: true,
	Columns: []string{"name", "namespace", "created_at", "updated_at", "deleted_at"},
	Aliases: map[string]string{
		"created": "created_at",
		"updated": "updated_at",
		"deleted": "deleted_at",
	},
	FieldMapper: map[string]func(ctx context.Context, id string) (any, error){
		"created_at": DateMapper,
		"updated_at": DateMapper,
		"deleted_at": DateMapper,
	},
}

func GetModelFromTable(table string) (QueryModel, error) {
	switch table {
	case models.ConfigItem{}.TableName():
		return ConfigQueryModel, nil
	case models.Component{}.TableName():
		return ComponentQueryModel, nil
	case models.Check{}.TableName():
		return CheckQueryModel, nil
	case models.Playbook{}.TableName():
		return PlaybookQueryModel, nil
	default:
		return QueryModel{}, fmt.Errorf("invalid table")
	}
}

// QueryModel.Apply will ignore these fields when converting to clauses
// as we modify the tx directly for them
var ignoreFieldsForClauses = []string{"sort", "offset", "limit", "labels", "config", "tags", "properties", "component_config_traverse"}

func (qm QueryModel) Apply(ctx context.Context, q grammar.QueryField, tx *gorm.DB) (*gorm.DB, []clause.Expression, error) {
	if tx == nil {
		tx = ctx.DB().Table(qm.Table)
	}
	clauses := []clause.Expression{}
	var err error

	if q.Field != "" {
		originalField := q.Field
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

		if mapper, ok := CommonFields[q.Field]; ok {
			tx, err = mapper(ctx, tx, val)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Invalid value for %s", q.Field)
			}
		}

		if mapper, ok := qm.Custom[q.Field]; ok {
			tx, err = mapper(ctx, tx, val)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Invalid value for %s", q.Field)
			}
		}

		for _, column := range qm.JSONMapColumns {
			// Keys in JSON fields are addressable as <column>.<key>
			// example: labels.cluster or tags.namespace
			if strings.HasPrefix(originalField, fmt.Sprintf("%s.", column)) {
				tx = JSONPathMapper(ctx, tx, column, q.Op, strings.TrimPrefix(originalField, column+"."), val)
				q.Field = column
			} else if strings.HasPrefix(q.Field, fmt.Sprintf("%s.", column)) {
				tx = JSONPathMapper(ctx, tx, column, q.Op, strings.TrimPrefix(q.Field, column+"."), val)
				q.Field = column
			}

			// Another way to search jsonb maps is to do an unkeyed lookup on the values
			// example: tags=default (matches tags={namespace: default})
			if originalField == column && (q.Op == grammar.Eq || q.Op == grammar.Neq) {
				tx = filterJSONColumnValues(tx, column, q.Op, val)
				q.Field = column
			}
		}

		if qm.HasProperties {
			column := "properties"
			if strings.HasPrefix(originalField, fmt.Sprintf("%s.", column)) {
				name := strings.TrimPrefix(originalField, fmt.Sprintf("%s.", column))
				tx = filterProperties(tx, q.Op, name, val)
				q.Field = column
			} else if originalField == column {
				tx = filterJSONColumnValues(tx, column, q.Op, val)
				q.Field = column
			}
		}

		if !slices.Contains(ignoreFieldsForClauses, q.Field) {
			if !slices.Contains(qm.Columns, q.Field) {
				return nil, nil, fmt.Errorf("query for column:%s in table:%s not supported", q.Field, qm.Table)
			}
			if c, err := q.ToClauses(); err != nil {
				return nil, nil, err
			} else {
				clauses = append(clauses, c...)
			}
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

func filterJSONColumnValues(tx *gorm.DB, column string, op grammar.QueryOperator, val string) *gorm.DB {
	if !slices.Contains([]grammar.QueryOperator{grammar.Eq, grammar.Neq}, op) {
		op = grammar.Eq
	}

	values := strings.Split(val, ",")

	switch column {
	case "tags":
		qf := grammar.QueryField{Field: "tags_values", FieldType: grammar.FieldTypeJsonbArray, Value: val, Op: op}
		clauses, err := qf.ToClauses()
		if err != nil {
			return nil
		}

		tx = tx.Clauses(clauses...)

	case "properties":
		qf := grammar.QueryField{Field: "properties_values", FieldType: grammar.FieldTypeJsonbArray, Value: val, Op: op}
		clauses, err := qf.ToClauses()
		if err != nil {
			return nil
		}

		tx = tx.Clauses(clauses...)

	default:
		subQueryCondition := lo.Ternary(op == grammar.Neq, "NOT EXISTS", "EXISTS")
		tx = tx.Where(fmt.Sprintf(`%s (
			SELECT 1 
			FROM jsonb_each_text(%s) 
			WHERE value IN ?
		)`, subQueryCondition, column), values)
	}

	return tx
}

func filterProperties(tx *gorm.DB, op grammar.QueryOperator, name string, text string) *gorm.DB {
	var subQueryCondition string
	switch op {
	case grammar.Neq:
		subQueryCondition = "NOT EXISTS"
	default:
		subQueryCondition = "EXISTS"
	}

	values := strings.Split(text, ",")
	subquery := fmt.Sprintf(`%s (
		SELECT 1
		FROM jsonb_array_elements(properties) AS prop
		WHERE prop->>'name' = ?
		AND prop->>'text' IN ?
	)`, subQueryCondition)

	tx = tx.Where(subquery, name, values)
	return tx
}
