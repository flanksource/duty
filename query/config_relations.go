package query

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type RelatedConfig struct {
	Relation      string              `json:"relation"`
	RelatedIDs    pq.StringArray      `json:"related_ids" gorm:"type:[]text"`
	ID            uuid.UUID           `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Tags          types.JSONStringMap `json:"tags"`
	Changes       int                 `json:"changes,omitempty"`
	Analysis      types.JSON          `json:"analysis,omitempty"`
	CostPerMinute *float64            `json:"cost_per_minute,omitempty"`
	CostTotal1d   *float64            `json:"cost_total_1d,omitempty"`
	CostTotal7d   *float64            `json:"cost_total_7d,omitempty"`
	CostTotal30d  *float64            `json:"cost_total_30d,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	DeletedAt     *time.Time          `json:"deleted_at"`
	AgentID       uuid.UUID           `json:"agent_id"`
	Status        *string             `json:"status" gorm:"default:null"`
	Ready         bool                `json:"ready"`
	Health        *models.Health      `json:"health"`
	Path          string              `json:"path"`
}

func (rc RelatedConfig) TemplateEnv() map[string]any {
	var deletedAt any
	if rc.DeletedAt != nil {
		deletedAt = rc.DeletedAt.Format(time.RFC3339Nano)
	}

	var status any
	if rc.Status != nil {
		status = *rc.Status
	}

	var health any
	if rc.Health != nil {
		health = *rc.Health
	}

	return map[string]any{
		"relation":        rc.Relation,
		"related_ids":     []string(rc.RelatedIDs),
		"id":              rc.ID.String(),
		"name":            rc.Name,
		"type":            rc.Type,
		"tags":            rc.Tags,
		"changes":         rc.Changes,
		"analysis":        rc.Analysis,
		"cost_per_minute": rc.CostPerMinute,
		"cost_total_1d":   rc.CostTotal1d,
		"cost_total_7d":   rc.CostTotal7d,
		"cost_total_30d":  rc.CostTotal30d,
		"created_at":      rc.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":      rc.UpdatedAt.Format(time.RFC3339Nano),
		"deleted_at":      deletedAt,
		"agent_id":        rc.AgentID.String(),
		"status":          status,
		"ready":           rc.Ready,
		"health":          health,
		"path":            rc.Path,
	}
}

type RelationQuery struct {
	ID             uuid.UUID
	Relation       RelationDirection
	Incoming       RelationType
	Outgoing       RelationType
	IncludeDeleted bool
	MaxDepth       *int
}

type RelationDirection string

const (
	All      RelationDirection = "all"
	Incoming RelationDirection = "incoming"
	Outgoing RelationDirection = "outgoing"
)

func (t RelationDirection) ToChangeDirection() ChangeRelationDirection {
	switch t {
	case All:
		return CatalogChangeRecursiveAll
	case Incoming:
		return CatalogChangeRecursiveUpstream
	case Outgoing:
		return CatalogChangeRecursiveDownstream
	}

	return CatalogChangeRecursiveNone
}

type RelationType string

const (
	Both RelationType = "both"
	Hard RelationType = "hard"
	Soft RelationType = "soft"
)

func GetRelatedConfigs(ctx context.Context, query RelationQuery) ([]RelatedConfig, error) {
	var relatedConfigs []RelatedConfig
	if query.MaxDepth == nil {
		query.MaxDepth = lo.ToPtr(5)
	}
	if query.Incoming == "" {
		query.Incoming = Both
	}
	if query.Outgoing == "" {
		query.Outgoing = Both
	}

	err := ctx.DB().Raw("SELECT * FROM related_configs_recursive(?, ?, ?, ?, ?, ?)",
		query.ID,
		query.Relation,
		query.IncludeDeleted,
		*query.MaxDepth,
		query.Incoming,
		query.Outgoing).Find(&relatedConfigs).Error

	return relatedConfigs, err
}
