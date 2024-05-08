package query

import (
	"strings"

	"gorm.io/gorm/clause"
)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
func ParseFilteringQuery(query string) (in []interface{}, notIN []interface{}, prefix, suffix []string) {
	if query == "" {
		return
	}

	items := strings.Split(query, ",")
	for _, item := range items {
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

func parseAndBuildFilteringQuery(query string, field string) []clause.Expression {
	var clauses []clause.Expression

	in, notIN, prefixes, suffixes := ParseFilteringQuery(query)
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

	return clauses
}
