package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Agent struct {
	ID          uuid.UUID           `json:"id,omitempty" gorm:"default:generate_ulid()"`
	Name        string              `json:"name"`
	Hostname    string              `json:"hostname,omitempty"`
	Description string              `json:"description,omitempty"`
	IP          string              `json:"ip,omitempty"`
	Version     string              `json:"version,omitempty"`
	Username    string              `json:"username,omitempty"`
	PersonID    *uuid.UUID          `json:"person_id,omitempty"`
	Properties  types.JSONStringMap `json:"properties,omitempty"`
	TLS         string              `json:"tls,omitempty"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at" time_format:"postgres_timestamp"`
	DeletedAt   *time.Time          `json:"deleted_at" time_format:"postgres_timestamp"`
	UpdatedAt   time.Time           `json:"updated_at" time_format:"postgres_timestamp"`

	// Cleanup when set to true will delete all the agent resources when the agent is deleted
	Cleanup bool `json:"cleanup"`

	// LastSeen is the timestamp the agent last sent a heartbeat
	LastSeen *time.Time `json:"last_seen,omitempty" time_format:"postgres_timestamp"`

	// LastReceived is the timestamp the agent last sent a push data
	LastReceived *time.Time `json:"last_received,omitempty" time_format:"postgres_timestamp"`
}

func (p *Agent) Save(db *gorm.DB) error {
	if p.ID != uuid.Nil {
		return db.Model(p).Clauses(
			clause.Returning{},
		).Save(p).Error
	}
	return db.Model(p).Clauses(
		clause.Returning{},
		clause.OnConflict{
			Columns:     []clause.Column{{Name: "name"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
			UpdateAll:   true,
		}).Create(p).Error
}

func (a Agent) Context() map[string]any {
	return map[string]any{
		"agent_id": a.ID,
	}
}

func (a Agent) TableName() string {
	return "agents"
}

func (a Agent) PK() string {
	return a.ID.String()
}

func (t Agent) AsMap(removeFields ...string) map[string]any {
	return asMap(t, removeFields...)
}
