package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/samber/lo"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/types"
)

const notificationSummaryPageSizeDefault = 50

type NotificationSendHistorySummaryRequest struct {
	GroupBy                 []string              `json:"groupBy"`
	Status                  types.MatchExpression `json:"status"` // matchItem
	ResourceType            string                `json:"resourceType"`
	Search                  string                `json:"search"` // search on resource name
	From                    string                `json:"from"`
	To                      string                `json:"to"`
	IncludeDeletedResources bool                  `json:"includeDeletedResources"`

	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`

	from *time.Time
	to   *time.Time
}

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

func (r *NotificationSendHistorySummaryRequest) baseSelectColumns() []string {
	// TODO: Must be dynamic based on groupBy
	return []string{
		"resource",
		"resource_type",
		"resource_tags",
		"resource_health",
		"resource_status",
		"resource_health_description",
		"first_observed",
		"created_at",
		"body",
		"status",
		"ROW_NUMBER() OVER (PARTITION BY resource ORDER BY created_at DESC) AS rn",
	}
}

func (r *NotificationSendHistorySummaryRequest) summarySelectColumns() []string {
	// TODO: Must be dynamic based on groupBy
	return []string{
		"resource",
		"MAX(CASE WHEN rn = 1 THEN resource_type END) AS resource_type",
		"ANY_VALUE(resource_tags) as resource_tags",
		"MAX(CASE WHEN rn = 1 THEN resource_health END) AS resource_health",
		"MAX(CASE WHEN rn = 1 THEN resource_status END) AS resource_status",
		"MAX(CASE WHEN rn = 1 THEN resource_health_description END) AS resource_health_description",
		"MIN(first_observed) AS first_observed",
		"MAX(created_at) AS last_seen",
		"COUNT(*) AS total",
		"MAX(CASE WHEN rn = 1 THEN body END) AS last_message",
		"COUNT(CASE WHEN status = 'sent' THEN 1 END) AS sent",
		"COUNT(CASE WHEN status = 'error' THEN 1 END) AS error",
		"COUNT(CASE WHEN status != 'error' AND status != 'sent' THEN 1 END) AS suppressed",
	}
}

func (r *NotificationSendHistorySummaryRequest) baseWhereClause() []clause.Expression {
	var clauses []clause.Expression
	if len(r.Status) > 0 {
		clause, _ := parseAndBuildFilteringQuery(string(r.Status), "status", false)
		clauses = append(clauses, clause...)
	}

	if r.from != nil {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "created_at"}, Value: *r.from})
	}

	if r.to != nil {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "created_at"}, Value: *r.to})
	}

	if r.ResourceType != "" {
		clause, _ := parseAndBuildFilteringQuery(string(r.ResourceType), "resource_type", false)
		clauses = append(clauses, clause...)
	}

	if !r.IncludeDeletedResources {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "resource->>'deleted_at'", Raw: true}, Value: nil})
	}

	if r.Search != "" {
		if !strings.Contains(r.Search, "*") {
			r.Search += "*" // prefix search by default
		}

		clause, _ := parseAndBuildFilteringQuery(r.Search, "resource->>'name'", true)
		clauses = append(clauses, clause...)
	}

	return clauses
}

func (r *NotificationSendHistorySummaryRequest) SetDefaults() {
	if len(r.GroupBy) == 0 {
		r.GroupBy = []string{"resource"}
	}

	if r.PageSize > notificationSummaryPageSizeDefault || r.PageSize < 1 {
		r.PageSize = notificationSummaryPageSizeDefault
	}

	if r.PageIndex < 0 {
		r.PageIndex = 0
	}
}

func (r *NotificationSendHistorySummaryRequest) getGroupByColumns() []string {
	var output []string
	for _, g := range r.GroupBy {
		switch g {
		case "resource", "resource_id", "resource_type", "resource_tags", "status", "source_event":
			output = append(output, g)
		default:
			logger.Debugf("unknown groupBy: %s", g)
		}
	}

	return output
}

type NotificationSendHistorySummaryResponse struct {
	Total   int64      `json:"total"`
	Results types.JSON `json:"results"`
}
