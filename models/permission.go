package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/types"
)

type PermissionGroup struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace,omitempty" gorm:"default:NULL"`
	Source    string     `json:"source"`
	Selectors types.JSON `json:"selectors"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty" gorm:"default:NULL"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (p PermissionGroup) GetNamespace() string {
	return p.Namespace
}

type PermissionSubjectType string

const (
	PermissionSubjectTypeCanary       PermissionSubjectType = "canary"
	PermissionSubjectTypeGroup        PermissionSubjectType = "group"
	PermissionSubjectTypeNotification PermissionSubjectType = "notification"
	PermissionSubjectTypePerson       PermissionSubjectType = "person"
	PermissionSubjectTypePlaybook     PermissionSubjectType = "playbook"
	PermissionSubjectTypeScraper      PermissionSubjectType = "scraper"
	PermissionSubjectTypeTeam         PermissionSubjectType = "team"
	PermissionSubjectTypeTopology     PermissionSubjectType = "topology"
)

func (p PermissionSubjectType) Pretty() api.Text {
	var icon string
	switch p {
	case PermissionSubjectTypePerson:
		icon = "👤"
	case PermissionSubjectTypeGroup, PermissionSubjectTypeTeam:
		icon = "👥"
	case PermissionSubjectTypePlaybook:
		icon = "📋"
	case PermissionSubjectTypeScraper:
		icon = "🔄"
	case PermissionSubjectTypeCanary:
		icon = "🐤"
	case PermissionSubjectTypeTopology:
		icon = "🗺️"
	case PermissionSubjectTypeNotification:
		icon = "🔔"
	default:
		icon = "•"
	}
	return clicky.Text(icon+" ", "text-gray-700").Append(string(p), "capitalize text-gray-700")
}

type Permission struct {
	ID          uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace,omitempty" gorm:"default:NULL"`
	Deny        bool       `json:"deny"`
	Description string     `json:"description"`
	Source      string     `json:"source"`
	Until       *time.Time `json:"until"`
	Error       *string    `json:"error,omitempty" gorm:"default:NULL"`

	// Action supports matchItem
	Action string `json:"action"`

	Subject     string                `json:"subject"`
	SubjectType PermissionSubjectType `json:"subject_type,omitempty"`
	// Deprecated: Use Subject
	PersonID *uuid.UUID `json:"person_id,omitempty"`
	// Deprecated: Use Subject
	NotificationID *uuid.UUID `json:"notification_id,omitempty"`
	// Deprecated: Use Subject
	TeamID *uuid.UUID `json:"team_id,omitempty"`

	CanaryID       *uuid.UUID `json:"canary_id,omitempty"`
	ComponentID    *uuid.UUID `json:"component_id,omitempty"`
	ConfigID       *uuid.UUID `json:"config_id,omitempty"`
	ConnectionID   *uuid.UUID `json:"connection_id,omitempty"`
	PlaybookID     *uuid.UUID `json:"playbook_id,omitempty"`
	Object         string     `json:"object,omitempty" gorm:"default:NULL"`
	ObjectSelector types.JSON `json:"object_selector,omitempty" gorm:"default:NULL"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty" gorm:"default:NULL"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedBy *uuid.UUID `json:"updated_by"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func (p Permission) PK() string {
	return p.ID.String()
}

func (p Permission) TableName() string {
	return "permissions"
}

func (p Permission) GetNamespace() string {
	return p.Namespace
}

func (t *Permission) Principal() string {
	if t.Subject != "" {
		return t.Subject
	}

	// NOTE: Person, team and notification ids are deprecated.
	// A single "subject" field is sufficient.
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

	if len(t.ObjectSelector) > 0 {
		rule = append(rule, fmt.Sprintf(`matchResourceSelector(r.obj, %q)`, string(t.ObjectSelector)))
	}

	if t.ComponentID != nil {
		rule = append(rule, fmt.Sprintf("str(r.obj.Component.ID) == %q", t.ComponentID.String()))
	}

	if t.ConfigID != nil {
		rule = append(rule, fmt.Sprintf("str(r.obj.Config.ID) == %q", t.ConfigID.String()))
	}

	if t.CanaryID != nil {
		rule = append(rule, fmt.Sprintf("str(r.obj.Canary.ID) == %q", t.CanaryID.String()))
	}

	if t.PlaybookID != nil {
		rule = append(rule, fmt.Sprintf("str(r.obj.Playbook.ID) == %q", t.PlaybookID.String()))
	}

	if t.ConnectionID != nil {
		rule = append(rule, fmt.Sprintf("str(r.obj.Connection.ID) == %q", t.ConnectionID.String()))
	}

	return strings.Join(rule, " && ")
}

func (t *Permission) GetObject() string {
	return lo.CoalesceOrEmpty(t.Object, "*")
}

func (t *Permission) Effect() string {
	if t.Deny {
		return "deny"
	}

	return "allow"
}
