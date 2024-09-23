package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CasbinRule struct {
	ID    int64  `gorm:"primaryKey;autoIncrement"`
	PType string `json:"ptype"`
	V0    string `json:"v0"`
	V1    string `json:"v1"`
	V2    string `json:"v2"`
	V3    string `json:"v3"`
	V4    string `json:"v4"`
	V5    string `json:"v5"`
}

type Permission struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Action      string     `json:"action"`
	CanaryID    *uuid.UUID `json:"canary_id,omitempty"`
	ComponentID *uuid.UUID `json:"component_id,omitempty"`
	ConfigID    *uuid.UUID `json:"config_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	Deny        bool       `json:"deny"`
	Description string     `json:"description"`
	PersonID    *uuid.UUID `json:"person_id,omitempty"`
	PlaybookID  *uuid.UUID `json:"playbook_id,omitempty"`
	TeamID      *uuid.UUID `json:"team_id,omitempty"`
	Until       *time.Time `json:"until"`
	UpdatedAt   *time.Time `json:"updated_at"`
	UpdatedBy   *uuid.UUID `json:"updated_by"`
}

func (t *Permission) Principal() string {
	var rule []string

	if t.PersonID != nil {
		rule = append(rule, fmt.Sprintf("r.sub.id == %s", t.PersonID.String()))
	} else if t.TeamID != nil {
		rule = append(rule, fmt.Sprintf("r.sub.id == %s", t.TeamID.String()))
	}

	if t.ComponentID != nil {
		rule = append(rule, fmt.Sprintf("r.component.id == %s", t.ComponentID.String()))
	}

	if t.ConfigID != nil {
		rule = append(rule, fmt.Sprintf("r.config.id == %s", t.ConfigID.String()))
	}

	if t.CanaryID != nil {
		rule = append(rule, fmt.Sprintf("r.canary.id == %s", t.CanaryID.String()))
	}

	if t.PlaybookID != nil {
		rule = append(rule, fmt.Sprintf("r.playbook.id == %s", t.PlaybookID.String()))
	}

	return strings.Join(rule, " && ")
}

func (t *Permission) Effect() string {
	if t.Deny {
		return "deny"
	}

	return "allow"
}
