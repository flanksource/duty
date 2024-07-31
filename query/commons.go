package query

import (
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
func ParseFilteringQuery(query string, decodeURL bool) (in []interface{}, notIN []interface{}, prefix, suffix []string, err error) {
	if query == "" {
		return
	}

	items := strings.Split(query, ",")
	for _, item := range items {
		if decodeURL {
			item, err = url.QueryUnescape(item)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to unescape query (%s): %v", item, err)
			}
		}

		if strings.HasPrefix(item, "!") {
			notIN = append(notIN, strings.TrimPrefix(item, "!"))
		} else if strings.HasPrefix(item, "*") {
			suffix = append(suffix, strings.TrimPrefix(item, "*"))
		} else if strings.HasSuffix(item, "*") {
			prefix = append(prefix, strings.TrimSuffix(item, "*"))
		} else {
			in = append(in, item)
		}
	}

	return
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
