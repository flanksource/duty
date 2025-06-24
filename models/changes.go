package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/types"
)

// CatalogChange represents the catalog_changes view
type CatalogChange struct {
	ID                uuid.UUID           `gorm:"primaryKey;column:id" json:"id"`
	ConfigID          uuid.UUID           `gorm:"column:config_id" json:"config_id"`
	Config            types.JSON          `gorm:"column:config" json:"config"`
	Name              *string             `gorm:"column:name" json:"name"`
	DeletedAt         *time.Time          `gorm:"column:deleted_at" json:"deleted_at"`
	Type              *string             `gorm:"column:type" json:"type"`
	Tags              types.JSONStringMap `gorm:"column:tags" json:"tags"`
	ExternalCreatedBy *string             `gorm:"column:external_created_by" json:"external_created_by"`
	CreatedAt         *time.Time          `gorm:"column:created_at" json:"created_at"`
	Severity          *string             `gorm:"column:severity" json:"severity"`
	ChangeType        string              `gorm:"column:change_type" json:"change_type"`
	Source            *string             `gorm:"column:source" json:"source"`
	Details           types.JSON          `json:"details,omitempty"`
	Summary           *string             `gorm:"column:summary" json:"summary"`
	CreatedBy         *uuid.UUID          `gorm:"column:created_by" json:"created_by"`
	Count             int                 `gorm:"column:count" json:"count"`
	FirstObserved     *time.Time          `gorm:"column:first_observed" json:"first_observed"`
	AgentID           *uuid.UUID          `gorm:"column:agent_id" json:"agent_id"`
}

func (c CatalogChange) GetID() string {
	return c.ID.String()
}

func (c CatalogChange) GetName() string {
	if c.Summary != nil && *c.Summary != "" {
		return *c.Summary
	}
	if c.Name != nil {
		return *c.Name
	}
	return ""
}

func (c CatalogChange) GetNamespace() string {
	if c.Tags != nil {
		if namespace, exists := c.Tags["namespace"]; exists {
			return namespace
		}
	}
	return ""
}

func (c CatalogChange) GetType() string {
	return c.ChangeType
}

func (c CatalogChange) TableName() string {
	return "catalog_changes"
}

func (c CatalogChange) PK() string {
	return c.ID.String()
}

func (c CatalogChange) AsMap(removeFields ...string) map[string]any {
	env := asMap(c, removeFields...)
	if c.Details != nil {
		var m map[string]any
		if err := json.Unmarshal(c.Details, &m); err != nil {
			return env
		}
		env["details"] = m
	}

	if c.Config != nil {
		var m map[string]any
		if err := json.Unmarshal(c.Config, &m); err != nil {
			return env
		}
		env["config"] = m
	}

	return env
}
