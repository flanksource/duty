package query

import (
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/samber/lo"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm/clause"
)

type NotificationSendHistorySummaryRequest struct {
	GroupBy                 []string              `json:"groupBy"`
	Status                  types.MatchExpression `json:"status"` // matchItem
	From                    string                `json:"from"`
	To                      string                `json:"to"`
	IncludeDeletedResources bool                  `json:"includeDeletedResources"`

	from *time.Time
	to   *time.Time
}

// TODO:
// func (r *NotificationSendHistorySummaryRequest) resourceDeletedClause() string {
// 	if r.IncludeDeletedResources {
// 		return ""
// 	}

// 	return "resource.deleted_at IS NULL"
// }

func (r *NotificationSendHistorySummaryRequest) Validate() error {
	if r.From != "" {
		if expr, err := datemath.Parse(r.From); err != nil {
			return fmt.Errorf("invalid from: %s", err)
		} else {
			r.from = lo.ToPtr(expr.Time())
		}
	}

	if r.To != "" {
		if expr, err := datemath.Parse(r.To); err != nil {
			return fmt.Errorf("invalid to: %s", err)
		} else {
			r.to = lo.ToPtr(expr.Time())
		}
	}

	return nil
}

func (r *NotificationSendHistorySummaryRequest) baseWhereClause() []clause.Expression {
	var clauses []clause.Expression
	if len(r.Status) > 0 {
		clause, _ := parseAndBuildFilteringQuery(string(r.Status), "status", false)
		clauses = append(clauses, clause...)
	}

	if r.from != nil {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "first_observed"}, Value: *r.from})
	}

	if r.to != nil {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "created_at"}, Value: *r.to})
	}

	return clauses
}

func (r *NotificationSendHistorySummaryRequest) SetDefaults() {
	if len(r.GroupBy) == 0 {
		r.GroupBy = []string{"resource", "resource_type"}
	}
}

func (r *NotificationSendHistorySummaryRequest) getGroupByColumns() []string {
	var output []string
	for _, g := range r.GroupBy {
		switch g {
		case "resource", "resource_id", "resource_type", "status", "source_event":
			output = append(output, g)
		default:
			logger.Debugf("unknown groupBy: %s", g)
		}
	}

	return output
}
