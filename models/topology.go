package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Topology represents the topologies database table
type Topology struct {
	ID        uuid.UUID           `gorm:"default:generate_ulid()"`
	AgentID   uuid.UUID           `json:"agent_id"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	Labels    types.JSONStringMap `json:"labels,omitempty"  gorm:"default:null"`
	Source    string              `json:"source"`
	Spec      types.JSON          `gorm:"default:null"`
	Schedule  *string             `json:"schedule,omitempty"`
	CreatedBy *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt *time.Time          `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:create"`
	UpdatedAt *time.Time          `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time          `json:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (t Topology) OnConflictClause() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_id"}, {Name: "name"}, {Name: "namespace"}},
		TargetWhere: clause.Where{
			Exprs: []clause.Expression{
				clause.And(
					clause.Eq{Column: "deleted_at"},
					clause.Expr{SQL: "agent_id = '00000000-0000-0000-0000-000000000000'::uuid", WithoutParentheses: true},
				),
			},
		},
		DoUpdates: clause.AssignmentColumns([]string{"labels", "spec"}),
	}
}

func (t Topology) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Topology
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Topology, _ int) DBTable { return i }), err
}

func (t Topology) GetAgentID() string {
	if t.AgentID == uuid.Nil {
		return ""
	}
	return t.AgentID.String()
}

func (t Topology) PK() string {
	return t.ID.String()
}

func (Topology) TableName() string {
	return "topologies"
}

func (t *Topology) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}

func (t *Topology) Save(db *gorm.DB) error {
	err := db.Clauses(Topology{}.OnConflictClause()).Create(t).Error
	return err
}

func (t Topology) GetNamespace() string {
	return t.Namespace
}
