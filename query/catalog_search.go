package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/pkg/kube/labels"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/timberio/go-datemath"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BaseCatalogSearch struct {
	CatalogID             string `query:"id" json:"id"`
	ConfigType            string `query:"config_type" json:"config_type"`
	IncludeDeletedConfigs bool   `query:"include_deleted_configs" json:"include_deleted_configs"`
	Depth                 int    `query:"depth" json:"depth"`
	Tags                  string `query:"tags" json:"tags"`
	AgentID               string `query:"agent_id" json:"agent_id"`
	From                  string `query:"from" json:"from"`
	To                    string `query:"to" json:"to"`
	PageSize              int    `query:"page_size" json:"page_size"`
	Page                  int    `query:"page" json:"page"`
	SortBy                string `query:"sort_by" json:"sort_by"`
	Recursive             ChangeRelationDirection `query:"recursive" json:"recursive"`
	Soft                  bool   `query:"soft" json:"soft"`

	sortOrder string
	configIDs []uuid.UUID
	FromTime  *time.Time `query:"-" json:"-"`
	ToTime    *time.Time `query:"-" json:"-"`
}

func (b *BaseCatalogSearch) SetDefaults() {
	if b.PageSize <= 0 {
		b.PageSize = 50
	}
	if b.Page <= 0 {
		b.Page = 1
	}
	if b.Depth <= 0 {
		b.Depth = 5
	}
	if b.Recursive == "" {
		b.Recursive = CatalogChangeRecursiveDownstream
	}
	if b.AgentID == "local" {
		b.AgentID = uuid.Nil.String()
	}
}

func (b *BaseCatalogSearch) Validate() error {
	if b.From != "" && b.FromTime == nil {
		expr, err := datemath.Parse(b.From)
		if err != nil {
			return fmt.Errorf("invalid 'from' param: %w", err)
		}
		t := expr.Time()
		b.FromTime = &t
	}
	if b.To != "" && b.ToTime == nil {
		expr, err := datemath.Parse(b.To)
		if err != nil {
			return fmt.Errorf("invalid 'to' param: %w", err)
		}
		t := expr.Time()
		b.ToTime = &t
	}
	if b.FromTime != nil && b.ToTime != nil && !b.FromTime.Before(*b.ToTime) {
		return fmt.Errorf("'from' must be before 'to'")
	}
	if b.AgentID != "" {
		if _, err := uuid.Parse(b.AgentID); err != nil {
			return fmt.Errorf("agent_id(%s) must either be a valid uuid or `local`", b.AgentID)
		}
	}
	return nil
}

func (b *BaseCatalogSearch) ResolveConfigIDs(ctx context.Context) ([]uuid.UUID, error) {
	if b.CatalogID == "" {
		return nil, nil
	}
	parts := strings.Split(b.CatalogID, ",")
	var ids []uuid.UUID
	allValid := true
	for _, p := range parts {
		if id, err := uuid.Parse(strings.TrimSpace(p)); err == nil {
			ids = append(ids, id)
		} else {
			allValid = false
			break
		}
	}
	if allValid && len(ids) > 0 {
		b.configIDs = ids
		return ids, nil
	}

	response, err := SearchResources(ctx, SearchResourcesRequest{
		Configs: []types.ResourceSelector{{Search: b.CatalogID, Cache: "no-cache"}},
		Limit:   200,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve catalog query %q: %w", b.CatalogID, err)
	}
	for _, c := range response.Configs {
		if id, err := uuid.Parse(c.ID); err == nil {
			ids = append(ids, id)
		}
	}
	b.configIDs = ids
	return ids, nil
}

func (b *BaseCatalogSearch) ConfigIDs() []uuid.UUID {
	return b.configIDs
}

func (b *BaseCatalogSearch) ApplyClauses() ([]clause.Expression, func(*gorm.DB) *gorm.DB) {
	var clauses []clause.Expression
	var tagsFn func(*gorm.DB) *gorm.DB

	if b.AgentID != "" {
		if c, err := parseAndBuildFilteringQuery(b.AgentID, "agent_id", false); err == nil {
			clauses = append(clauses, c...)
		}
	}
	if b.ConfigType != "" {
		if c, err := parseAndBuildFilteringQuery(b.ConfigType, "type", false); err == nil {
			clauses = append(clauses, c...)
		}
	}
	if b.FromTime != nil {
		clauses = append(clauses, clause.Gte{Column: clause.Column{Name: "created_at"}, Value: *b.FromTime})
	}
	if b.ToTime != nil {
		clauses = append(clauses, clause.Lte{Column: clause.Column{Name: "created_at"}, Value: *b.ToTime})
	}
	if !b.IncludeDeletedConfigs {
		clauses = append(clauses, clause.Eq{Column: clause.Column{Name: "deleted_at"}, Value: nil})
	}
	if b.Tags != "" {
		if parsedLabelSelector, err := labels.Parse(b.Tags); err == nil {
			requirements, _ := parsedLabelSelector.Requirements()
			tagsFn = func(db *gorm.DB) *gorm.DB {
				for _, r := range requirements {
					db = jsonColumnRequirementsToSQLClause(db, "tags", r)
				}
				return db
			}
		}
	}
	return clauses, tagsFn
}

func (b *BaseCatalogSearch) String() string {
	s := ""
	if b.AgentID != "" {
		s += fmt.Sprintf("agent=%s ", b.AgentID)
	}
	if b.CatalogID != "" {
		s += fmt.Sprintf("id=%s ", b.CatalogID)
	}
	if b.ConfigType != "" {
		s += fmt.Sprintf("config_type=%s ", b.ConfigType)
	}
	if b.Tags != "" {
		s += fmt.Sprintf("tags=%s ", b.Tags)
	}
	if b.From != "" {
		s += fmt.Sprintf("from=%s ", b.From)
	}
	if b.To != "" {
		s += fmt.Sprintf("to=%s ", b.To)
	}
	if b.Recursive != "" {
		s += fmt.Sprintf("recursive=%s ", b.Recursive)
	}
	return strings.TrimSpace(s)
}
