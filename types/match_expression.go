package types

import (
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/samber/lo"
	"gorm.io/gorm"
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
func (t MatchExpressions) SQLClause(db *gorm.DB, columnName string) (string, []any, error) {
	if len(t) == 0 {
		return "", nil, nil
	}

	query := db.Session(&gorm.Session{DryRun: true})

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
					query = query.Where(fmt.Sprintf("%s NOT LIKE ?", columnName), likePattern)
				} else if after, ok := strings.CutPrefix(after, "*"); ok {
					likePattern := "%" + after
					query = query.Where(fmt.Sprintf("%s NOT LIKE ?", columnName), likePattern)
				} else {
					query = query.Where(fmt.Sprintf("%s <> ?", columnName), after)
				}
			} else {
				if strings.HasSuffix(pattern, "*") {
					likePattern := strings.TrimSuffix(pattern, "*") + "%"
					query = query.Where(fmt.Sprintf("%s LIKE ?", columnName), likePattern)
				} else if after, ok := strings.CutPrefix(pattern, "*"); ok {
					likePattern := "%" + after
					query = query.Where(fmt.Sprintf("%s LIKE ?", columnName), likePattern)
				} else {
					query = query.Where(fmt.Sprintf("%s = ?", columnName), pattern)
				}
			}
		}
	}

	// Build a dummy query to extract the WHERE clause
	var output struct{}
	stmt := query.Select("1").Table("dummy_model").Find(&output).Statement

	whereClause := strings.TrimPrefix(stmt.SQL.String(), `SELECT 1 FROM "dummy_model"`)
	whereClause = strings.TrimSpace(whereClause)
	whereClause = strings.TrimPrefix(whereClause, "WHERE ")
	whereClause = strings.TrimSpace(whereClause)

	// Replace database-specific placeholders with generic ? placeholders
	// Postgres driver uses $1, $2, etc. placeholders
	for i := len(stmt.Vars); i > 0; i-- {
		whereClause = strings.Replace(whereClause, fmt.Sprintf("$%d", i), "?", 1)
	}

	return whereClause, stmt.Vars, nil
}
