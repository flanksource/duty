package query

import (
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"
)

type RelatedConfig struct {
	Relation      string              `json:"relation"`
	RelatedIDs    pq.StringArray      `json:"related_ids" gorm:"type:[]text"`
	ID            uuid.UUID           `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Tags          types.JSONStringMap `json:"tags"`
	Changes       types.JSON          `json:"changes,omitempty"`
	Analysis      types.JSON          `json:"analysis,omitempty"`
	CostPerMinute *float64            `json:"cost_per_minute,omitempty"`
	CostTotal1d   *float64            `json:"cost_total_1d,omitempty"`
	CostTotal7d   *float64            `json:"cost_total_7d,omitempty"`
	CostTotal30d  *float64            `json:"cost_total_30d,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	AgentID       uuid.UUID           `json:"agent_id"`
	Status        *string             `json:"status" gorm:"default:null"`
	Ready         bool                `json:"ready"`
	Health        *models.Health      `json:"health"`
	Path          string              `json:"path"`
}

type RelationType string
type RelationDirection string

type RelationQuery struct {
	ID             uuid.UUID
	Relation       RelationDirection
	Incoming       RelationType
	Outgoing       RelationType
	IncludeDeleted bool
	MaxDepth       *int
}

const (
	Incoming RelationDirection = "incoming"
	Outgoing RelationDirection = "outgoing"
	Both     RelationType      = "both"
	Hard     RelationType      = "hard"
	Soft     RelationType      = "soft"
	All      RelationDirection = "all"
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
