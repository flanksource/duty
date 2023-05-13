package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Agent struct {
	ID          uuid.UUID           `json:"id,omitempty" gorm:"default:generate_ulid()"`
	Name        string              `json:"name,omitempty"`
	Hostname    string              `json:"hostname,omitempty"`
	Description string              `json:"description,omitempty"`
	IP          string              `json:"ip,omitempty"`
	Version     string              `json:"version,omitempty"`
	Username    string              `json:"username,omitempty"`
	PersonID    *uuid.UUID          `json:"person_id,omitempty"`
	Properties  types.JSONStringMap `json:"properties,omitempty"`
	TLS         string              `json:"tls,omitempty"`
	CreatedBy   *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at,omitempty"`
}
