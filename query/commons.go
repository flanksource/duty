package query

import (
	"fmt"
	"strings"
)

var LocalFilter = "deleted_at is NULL AND agent_id = '00000000-0000-0000-0000-000000000000' OR agent_id IS NULL"

// ParseFilteringQuery parses a filtering query string.
// It returns four slices: 'in', 'notIN', 'prefix', and 'suffix'.
func ParseFilteringQuery(query string) (in, notIN, prefix, suffix []string) {
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

func parseAndBuildFilteringQuery(query string, field string) ([]string, map[string]any) {
	var clauses []string
	var args = map[string]any{}

	in, notIN, prefixes, suffixes := ParseFilteringQuery(query)
	if len(in) > 0 {
		clauses = append(clauses, fmt.Sprintf("%s IN @field_in", field))
		args["field_in"] = in
	}

	if len(notIN) > 0 {
		clauses = append(clauses, fmt.Sprintf("%s NOT IN @field_not_in", field))
		args["field_not_in"] = notIN
	}

	for i, p := range prefixes {
		clauses = append(clauses, fmt.Sprintf("%s LIKE @%s_prefix_%d", field, field, i))
		args[fmt.Sprintf("prefix_%d", i)] = fmt.Sprintf("%s%%", p)
	}

	for i, s := range suffixes {
		clauses = append(clauses, fmt.Sprintf("%s LIKE @%s_suffix_%d", field, field, i))
		args[fmt.Sprintf("suffix_%d", i)] = fmt.Sprintf("%%%s", s)
	}

	return clauses, args
}
