package query

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

type expressions struct {
	In     []interface{}
	Prefix []string
	Suffix []string
}

type Expressions []clause.Expression

// postgrestValues returns ["a", "b", "c"] as `"a","b","c"`
func postgrestValues(val []any) string {
	return strings.Join(lo.Map(val, func(s any, i int) string {
		return fmt.Sprintf(`"%s"`, s)
	}), ",")
}

func (query FilteringQuery) AppendPostgrest(key string,
	queryParam url.Values) {

	if len(query.In) > 0 {
		queryParam.Add(key, fmt.Sprintf("in.(%s)", postgrestValues(query.In)))
	}

	if len(query.Not.In) > 0 {
		queryParam.Add(key, fmt.Sprintf("not.in.(%s)", postgrestValues(query.Not.In)))
	}

	for _, p := range query.Prefix {
		queryParam.Add(key, fmt.Sprintf("like.%s*", p))
	}

	for _, p := range query.Suffix {
		queryParam.Add(key, fmt.Sprintf("like.*%s", p))
	}

}

func (e expressions) ToExpression(field string) []clause.Expression {

	var clauses []clause.Expression
	if len(e.In) > 0 {
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
	exprs := fq.expressions.ToExpression(field)
	not := clause.Not(fq.Not.ToExpression(field)...)
	return append(exprs, not)
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

		var q expressions
		q = result.expressions
		if strings.HasPrefix(item, "!") {
			q = result.Not
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

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
func ParseFilteringQuery(query string, decodeURL bool) (in []interface{}, notIN []interface{}, prefix, suffix []string, err error) {
	if query == "" {
		return
	}

	q, err := ParseFilteringQueryV2(query, decodeURL)

	if err != nil {
		return nil, nil, nil, nil, err
	}
	return q.In, q.Not.In, q.Prefix, q.Suffix, nil
}

func parseAndBuildFilteringQuery(query, field string, decodeURL bool) ([]clause.Expression, error) {
	in, notIN, prefixes, suffixes, err := ParseFilteringQuery(query, decodeURL)
	if err != nil {
		return nil, err
	}

	var clauses []clause.Expression
	if len(in) > 0 {
		clauses = append(clauses, clause.IN{Column: clause.Column{Name: field}, Values: in})
	}

	if len(notIN) > 0 {
		clauses = append(clauses, clause.NotConditions{
			Exprs: []clause.Expression{clause.IN{Column: clause.Column{Name: field}, Values: notIN}},
		})
	}

	for _, p := range prefixes {
		clauses = append(clauses, clause.Like{
			Column: clause.Column{Name: field},
			Value:  p + "%",
		})
	}

	for _, s := range suffixes {
		clauses = append(clauses, clause.Like{
			Column: clause.Column{Name: field},
			Value:  "%" + s,
		})
	}

	return clauses, nil
}

func OrQueries(db *gorm.DB, queries ...*gorm.DB) *gorm.DB {
	if len(queries) == 0 {
		return db
	}

	if len(queries) == 1 {
		return db.Where(queries[0])
	}

	union := queries[0]
	for i, q := range queries {
		if i == 0 {
			continue
		}

		union = union.Or(q)
	}

	return db.Where(union)
}
