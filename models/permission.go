package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Permission struct {
	ID             uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Action         string     `json:"action"`
	Object         string     `json:"object"`
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty"`
	CanaryID       *uuid.UUID `json:"canary_id,omitempty"`
	NotificationID *uuid.UUID `json:"notification_id,omitempty"`
	ComponentID    *uuid.UUID `json:"component_id,omitempty"`
	ConfigID       *uuid.UUID `json:"config_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	Deny           bool       `json:"deny"`
	Description    string     `json:"description"`
	PersonID       *uuid.UUID `json:"person_id,omitempty"`
	PlaybookID     *uuid.UUID `json:"playbook_id,omitempty"`
	TeamID         *uuid.UUID `json:"team_id,omitempty"`
	Source         string     `json:"source"`
	Until          *time.Time `json:"until"`
	UpdatedAt      *time.Time `json:"updated_at"`
	UpdatedBy      *uuid.UUID `json:"updated_by"`

	// List of agent ids whose configs/components are accessible to a person when RLS is enabled
	Agents pq.StringArray `json:"agents,omitempty" gorm:"type:[]text"`

	// List of config/component tags a person is allowed access to when RLS is enabled
	Tags types.JSONStringMap `json:"tags,omitempty"`
}

func (t *Permission) Principal() string {
	if t.PersonID != nil {
		return t.PersonID.String()
	}

	if t.TeamID != nil {
		return t.TeamID.String()
	}

	if t.NotificationID != nil {
		return t.NotificationID.String()
	}

	return ""
}

func (t *Permission) Condition() string {
	var rule []string

	if t.ComponentID != nil {
		rule = append(rule, fmt.Sprintf("r.obj.component != undefined && r.obj.component.id == %q", t.ComponentID.String()))
	}

	if t.ConfigID != nil {
		rule = append(rule, fmt.Sprintf("r.obj.config != undefined && r.obj.config.id == %q", t.ConfigID.String()))
	}

	if t.CanaryID != nil {
		rule = append(rule, fmt.Sprintf("r.obj.canary != undefined && r.obj.canary.id == %q", t.CanaryID.String()))
	}

	if t.PlaybookID != nil {
		rule = append(rule, fmt.Sprintf("r.obj.playbook != undefined && r.obj.playbook.id == %q", t.PlaybookID.String()))
	}

	if len(t.Agents) > 0 || len(t.Tags) > 0 {
		var agents []string
		for _, agentID := range t.Agents {
			agents = append(agents, fmt.Sprintf("'%s'", agentID))
		}

		rule = append(rule, fmt.Sprintf(`"matchPerm(r.obj, (%s), '%s')"`, strings.Join(agents, ","), collections.SortedMap(t.Tags)))
	}

	return strings.Join(rule, " && ")
}

func (t *Permission) Effect() string {
	if t.Deny {
		return "deny"
	}

	return "allow"
}
