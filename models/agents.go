package models

import (
	"encoding/json"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
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
	UpdatedAt   time.Time           `json:"updated_at" time_format:"postgres_timestamp"`
}

func (t Agent) AsMap(removeFields ...string) map[string]any {
	m := make(map[string]any)
	b, _ := json.Marshal(&t)
	if err := json.Unmarshal(b, &m); err != nil {
		return m
	}

	for _, field := range removeFields {
		delete(m, field)
	}

	return m
}
