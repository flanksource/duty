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

// Role is a named set of rules, each allowing or denying an action on the resources selected by a scope,
// optionally on the targets selected by another scope.
// It grants nothing on its own: a RoleBinding grants it to subjects.
//
// Rules are stored only on the role. They are compiled into casbin policies,
// once per binding of the role, when the policy is loaded.
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"default:generate_ulid()"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace,omitempty"`
	Description string    `json:"description,omitempty" gorm:"default:NULL"`
	Source      string    `json:"source"`

	// Rules is the JSON list of rules as declared on the role
	Rules types.JSON `json:"rules"`

	// Error says why the role isn't in effect. Nil when it's valid.
	Error *string `json:"error,omitempty" gorm:"default:NULL"`

	// ErrorReason is a machine readable reason for Error, e.g. RowLevelSecurityRequired.
	ErrorReason *string `json:"error_reason,omitempty" gorm:"default:NULL"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty" gorm:"default:NULL"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (r Role) TableName() string {
	return "roles"
}

func (r Role) GetNamespace() string {
	return r.Namespace
}

// RoleBinding grants a role (in the same namespace) to many subjects,
// optionally narrowing the role's allow rules with constraints.
type RoleBinding struct {
	ID          uuid.UUID `json:"id" gorm:"default:generate_ulid()"`
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace,omitempty"`
	Description string    `json:"description,omitempty" gorm:"default:NULL"`
	Source      string    `json:"source"`

	// Role is the name of the bound role, in the binding's namespace
	Role string `json:"role"`

	// Constraints is the JSON list of constraints, each narrowing one rule of the role
	Constraints types.JSON `json:"constraints"`

	// Subjects is a JSON object of subject selectors
	Subjects types.JSON `json:"subjects"`

	// Error says why the role binding isn't in effect. Nil when it's valid.
	Error *string `json:"error,omitempty" gorm:"default:NULL"`

	// ErrorReason is a machine readable reason for Error, e.g. RoleNotFound.
	ErrorReason *string `json:"error_reason,omitempty" gorm:"default:NULL"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty" gorm:"default:NULL"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"<-:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (r RoleBinding) TableName() string {
	return "role_bindings"
}

func (r RoleBinding) GetNamespace() string {
	return r.Namespace
}

// Principal returns the casbin subject the role's rules are filed under for this binding.
func (r RoleBinding) Principal() string {
	return BindingPrincipal(r.Namespace, r.Name)
}

// BindingPrincipal returns the casbin subject for a role binding.
func BindingPrincipal(namespace, name string) string {
	return BindingPrincipalPrefix + namespace + "/" + name
}

// FederatedPrincipal returns the casbin subject for a person authenticated by an external identity provider.
func FederatedPrincipal(personID string) string {
	return FederatedPrincipalPrefix + personID
}

const (
	BindingPrincipalPrefix   = "binding:"
	FederatedPrincipalPrefix = "federated:"
)

type PermissionSubjectType string

const (
	PermissionSubjectTypeCanary       PermissionSubjectType = "canary"
	PermissionSubjectTypeGroup        PermissionSubjectType = "group"
	PermissionSubjectTypeNotification PermissionSubjectType = "notification"
	PermissionSubjectTypePerson       PermissionSubjectType = "person"
	PermissionSubjectTypePlaybook     PermissionSubjectType = "playbook"
	PermissionSubjectTypePlugin       PermissionSubjectType = "plugin"
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
	case PermissionSubjectTypePlugin:
		icon = "🧩"
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
