package grammar

import (
	"fmt"

	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/selection"
)

type QueryOperator string

const (
	Eq  QueryOperator = "="
	Neq QueryOperator = "!="

	Gt        QueryOperator = ">"
	Lt        QueryOperator = "<"
	In        QueryOperator = "in"
	NotIn     QueryOperator = "notin"
	Exists    QueryOperator = "exists"
	NotExists QueryOperator = "notexists"
)

func (op QueryOperator) ToSelectionOperator() selection.Operator {
	switch op {
	case Eq:
		return selection.Equals
	case Neq:
		return selection.NotEquals
	case In:
		return selection.In
	case NotIn:
		return selection.NotIn
	case Exists:
		return selection.Exists
	case NotExists:
		return selection.DoesNotExist
	default:
		return selection.Equals
	}
}

type QueryField struct {
	Field     string        `json:"field,omitempty"`
	FieldType FieldType     `json:"fieldType,omitempty"`
	Value     interface{}   `json:"value,omitempty"`
	Op        QueryOperator `json:"op,omitempty"`
	Not       bool          `json:"not,omitempty"`
	Fields    []*QueryField `json:"fields,omitempty"`
}

func (q QueryField) ToClauses() ([]clause.Expression, error) {
	val := fmt.Sprint(q.Value)

	filters, err := ParseFilteringQueryV2(val, false)
	if err != nil {
		return nil, err
	}

	var clauses []clause.Expression
	switch q.Op {
	case Eq:
		clauses = append(clauses, filters.ToExpression(q.Field, q.FieldType)...)
	case Neq:
		clauses = append(clauses, clause.Not(filters.ToExpression(q.Field, q.FieldType)...))
	case Lt:
		clauses = append(clauses, clause.Lt{Column: q.Field, Value: q.Value})
	case Gt:
		clauses = append(clauses, clause.Gt{Column: q.Field, Value: q.Value})
	default:
		return nil, fmt.Errorf("invalid operator: %s", q.Op)
	}

	return clauses, nil
}
