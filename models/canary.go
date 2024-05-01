package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Canary struct {
	ID          uuid.UUID           `json:"id" yaml:"id" gorm:"default:generate_ulid()"`
	Name        string              `json:"name" yaml:"name"`
	Namespace   string              `json:"namespace" yaml:"namespace"`
	AgentID     uuid.UUID           `json:"agent_id" yaml:"agent_id"`
	Spec        types.JSON          `json:"spec" yaml:"spec"`
	Labels      types.JSONStringMap `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations types.JSONStringMap `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Source      string              `json:"source,omitempty" yaml:"source,omitempty"`
	Checks      types.JSONStringMap `gorm:"-" json:"checks,omitempty" yaml:"checks,omitempty"`
	CreatedAt   time.Time           `json:"created_at" yaml:"created_at" time_format:"postgres_timestamp" gorm:"<-:create"`
	UpdatedAt   *time.Time          `json:"updated_at" yaml:"updated_at" time_format:"postgres_timestamp" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (t Canary) ConflictClause() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_id"}, {Name: "name"}, {Name: "namespace"}, {Name: "source"}},
		TargetWhere: clause.Where{
			Exprs: []clause.Expression{
				clause.Or(
					clause.Eq{Column: "deleted_at", Value: gorm.Expr("NULL")},
					clause.Not(clause.Eq{Column: "agent_id", Value: uuid.Nil.String()}),
				),
			},
		},
		DoUpdates: clause.AssignmentColumns([]string{"labels", "spec"}),
	}
}

func (t Canary) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Canary
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Canary, _ int) DBTable { return i }), err
}

func (c Canary) GetCheckID(checkName string) string {
	return c.Checks[checkName]
}

func (c Canary) PK() string {
	return c.ID.String()
}

func (c Canary) TableName() string {
	return "canaries"
}

func (c Canary) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}
