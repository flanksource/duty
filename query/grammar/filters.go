package grammar

import (
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FieldType int

const (
	FieldTypeText FieldType = iota
	FieldTypeJsonbArray
)

type expressions struct {
	In     []any
	Prefix []string
	Suffix []string
	Glob   []string
}

type Expressions []clause.Expression

func (e expressions) ToExpression(field string, fieldType FieldType) []clause.Expression {
	if fieldType == FieldTypeJsonbArray {
		return e.jsonbListFieldExpression(field)
	}

	return e.textFieldExpression(field)
}

func (e expressions) jsonbListFieldExpression(field string) []clause.Expression {
	var clauses []clause.Expression

	if len(e.In) > 0 {
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf(`%s ? ?`, field),
			Vars: []any{gorm.Expr("?"), e.In},
		})
	}

	for _, g := range e.Glob {
		regexp := fmt.Sprintf(".*%s.*", g)
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf(`jsonb_path_exists(?, '$[*] ? (@ like_regex "%s")')`, regexp),
			Vars: []any{clause.Column{Name: field}, gorm.Expr("?")},
		})
	}

	for _, p := range e.Prefix {
		regexp := fmt.Sprintf("^%s.*", p)
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf(`jsonb_path_exists(?, '$[*] ? (@ like_regex "%s")')`, regexp),
			Vars: []any{clause.Column{Name: field}, gorm.Expr("?")},
		})
	}

	for _, s := range e.Suffix {
		regexp := fmt.Sprintf(".*%s$", s)
		clauses = append(clauses, clause.Expr{
			SQL:  fmt.Sprintf(`jsonb_path_exists(?, '$[*] ? (@ like_regex "%s")')`, regexp),
			Vars: []any{clause.Column{Name: field}, gorm.Expr("?")},
		})
	}

	return clauses
}

func (e expressions) textFieldExpression(field string) []clause.Expression {
	var clauses []clause.Expression

	if len(e.In) == 1 {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: field}, Value: e.In[0]})
	} else if len(e.In) > 1 {
		clauses = append(clauses, clause.IN{Column: clause.Column{Name: field}, Values: e.In})
	}

	for _, p := range e.Prefix {
		clauses = append(clauses, clause.Like{
			Column: clause.Column{Name: field},
			Value:  p + "%",
		})
	}

	for _, g := range e.Glob {
		clauses = append(clauses, clause.Like{
			Column: clause.Column{Raw: true, Name: field},
			Value:  "%" + g + "%",
		})
	}

	for _, s := range e.Suffix {
		clauses = append(clauses, clause.Like{
			Column: clause.Column{Name: field},
			Value:  "%" + s,
		})
	}

	return clauses
}

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
type FilteringQuery struct {
	expressions
	Not expressions
}

func (fq *FilteringQuery) ToExpression(field string, fieldType FieldType) []clause.Expression {
	var clauses []clause.Expression
	if len(fq.expressions.ToExpression(field, fieldType)) > 0 {
		clauses = append(clauses, fq.expressions.ToExpression(field, fieldType)...)
	}
	if len(fq.Not.ToExpression(field, fieldType)) > 0 {
		clauses = append(clauses, clause.Not(fq.Not.ToExpression(field, fieldType)...))
	}
	return clauses
}

func ParseFilteringQueryV2(query string, decodeURL bool) (FilteringQuery, error) {
	result := FilteringQuery{}
	if query == "" {
		return result, nil
	}

	for item := range strings.SplitSeq(query, ",") {
		if decodeURL {
			var err error
			item, err = url.QueryUnescape(item)
			if err != nil {
				return FilteringQuery{}, fmt.Errorf("failed to unescape query (%s): %v", item, err)
			}
		}

		q := &result.expressions
		if strings.HasPrefix(item, "!") {
			q = &result.Not
			item = strings.TrimPrefix(item, "!")
		}

		if strings.HasPrefix(item, "*") && strings.HasSuffix(item, "*") {
			q.Glob = append(q.Glob, strings.Trim(item, "*"))
		} else if strings.HasPrefix(item, "*") {
			q.Suffix = append(q.Suffix, strings.TrimPrefix(item, "*"))
		} else if strings.HasSuffix(item, "*") {
			q.Prefix = append(q.Prefix, strings.TrimSuffix(item, "*"))
		} else {
			q.In = append(q.In, item)
		}
	}

	return result, nil
}
