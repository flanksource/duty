package grammar

import (
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm/clause"
)

type expressions struct {
	In     []interface{}
	Prefix []string
	Suffix []string
}

type Expressions []clause.Expression

func (e expressions) ToExpression(field string) []clause.Expression {
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

func (fq *FilteringQuery) ToExpression(field string) []clause.Expression {
	var clauses []clause.Expression
	if len(fq.expressions.ToExpression(field)) > 0 {
		clauses = append(clauses, fq.expressions.ToExpression(field)...)
	}
	if len(fq.Not.ToExpression(field)) > 0 {
		clauses = append(clauses, clause.Not(fq.Not.ToExpression(field)...))
	}
	return clauses
}

func ParseFilteringQueryV2(query string, decodeURL bool) (FilteringQuery, error) {
	result := FilteringQuery{}
	if query == "" {
		return result, nil
	}

	items := strings.Split(query, ",")
	for _, item := range items {
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
		if strings.HasPrefix(item, "*") {
			q.Suffix = append(q.Suffix, strings.TrimPrefix(item, "*"))
		} else if strings.HasSuffix(item, "*") {
			q.Prefix = append(q.Prefix, strings.TrimSuffix(item, "*"))
		} else {
			q.In = append(q.In, item)
		}

	}

	return result, nil
}
