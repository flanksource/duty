package types

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/flanksource/commons/collections"
	"github.com/samber/lo"
)

// MatchExpression uses MatchItems
type MatchExpression string

func (t MatchExpression) Match(item string) bool {
	return collections.MatchItems(item, strings.Split(string(t), ",")...)
}

func (t *MatchExpression) Add(item string) {
	if *t == "" {
		*t = MatchExpression(item)
	} else {
		*t = MatchExpression(fmt.Sprintf("%s,%s", *t, item))
	}
}

type MatchExpressions []MatchExpression

func (t MatchExpressions) Match(item string) bool {
	return collections.MatchItems(item, lo.Map(t, func(x MatchExpression, _ int) string { return string(x) })...)
}

// SQLClause converts MatchExpressions to SQL WHERE conditions with (?) placeholders
//
// Example: expr="!Get*,!List*" -> "WHERE (column_name NOT LIKE ? AND column_name NOT LIKE ?)" (GET%, LIST%)
func (t MatchExpressions) SQLClause(columnName string) (string, []any, error) {
	if len(t) == 0 {
		return "", nil, nil
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
	conditions := squirrel.And{}

	for _, expr := range t {
		patterns := strings.SplitSeq(string(expr), ",")
		for pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}

			if after, ok := strings.CutPrefix(pattern, "!"); ok {
				if strings.HasSuffix(after, "*") {
					likePattern := strings.TrimSuffix(after, "*") + "%"
					conditions = append(conditions, squirrel.NotLike{columnName: likePattern})
				} else if after, ok := strings.CutPrefix(after, "*"); ok {
					likePattern := "%" + after
					conditions = append(conditions, squirrel.NotLike{columnName: likePattern})
				} else {
					conditions = append(conditions, squirrel.NotEq{columnName: after})
				}
			} else {
				if strings.HasSuffix(pattern, "*") {
					likePattern := strings.TrimSuffix(pattern, "*") + "%"
					conditions = append(conditions, squirrel.Like{columnName: likePattern})
				} else if after, ok := strings.CutPrefix(pattern, "*"); ok {
					likePattern := "%" + after
					conditions = append(conditions, squirrel.Like{columnName: likePattern})
				} else {
					conditions = append(conditions, squirrel.Eq{columnName: pattern})
				}
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}

	query := psql.Select("1").Where(conditions)
	sql, args, err := query.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build SQL conditions: %w", err)
	}

	whereClause := strings.TrimPrefix(sql, "SELECT 1 WHERE ")
	return whereClause, args, nil
}
