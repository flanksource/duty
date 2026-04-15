package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GroupType is implemented by all typed group detail structs
// for the ChangeGroup.Details field.
type GroupType interface {
	Kind() string
}

// Change group type constants.
const (
	GroupTypeStartup             = "Startup/v1"
	GroupTypeDeployment          = "Deployment/v1"
	GroupTypePromotion           = "Promotion/v1"
	GroupTypeTemporaryPermission = "TemporaryPermission/v1"
	GroupTypeIncidentResponse    = "IncidentResponse/v1"
	GroupTypeCustom              = "Custom/v1"
)

// ConfigChangeGroupDetailsSchema is a union type used for JSON schema generation.
// It is never instantiated at runtime.
type ConfigChangeGroupDetailsSchema struct {
	Startup             *StartupGroup             `json:"Startup/v1,omitempty"`
	Deployment          *DeploymentGroup          `json:"Deployment/v1,omitempty"`
	Promotion           *PromotionGroup           `json:"Promotion/v1,omitempty"`
	TemporaryPermission *TemporaryPermissionGroup `json:"TemporaryPermission/v1,omitempty"`
	IncidentResponse    *IncidentResponseGroup    `json:"IncidentResponse/v1,omitempty"`
	Custom              *CustomGroup              `json:"Custom/v1,omitempty"`
}

type StartupGroup struct {
	ConfigID     uuid.UUID `json:"config_id"`
	Reason       string    `json:"reason,omitempty"`
	RestartCount int       `json:"restart_count,omitempty"`
}

func (g StartupGroup) Kind() string { return GroupTypeStartup }
func (g StartupGroup) MarshalJSON() ([]byte, error) {
	type raw StartupGroup
	return marshalWithKind(g.Kind(), raw(g))
}

type DeploymentGroup struct {
	Image           string      `json:"image,omitempty"`
	Version         string      `json:"version,omitempty"`
	Commit          string      `json:"commit,omitempty"`
	Strategy        string      `json:"strategy,omitempty"`
	TargetConfigIDs []uuid.UUID `json:"target_config_ids,omitempty" merge:"append"`
}

func (g DeploymentGroup) Kind() string { return GroupTypeDeployment }
func (g DeploymentGroup) MarshalJSON() ([]byte, error) {
	type raw DeploymentGroup
	return marshalWithKind(g.Kind(), raw(g))
}

type PromotionGroup struct {
	FromEnvironment     string      `json:"from_environment,omitempty"`
	ToEnvironment       string      `json:"to_environment,omitempty"`
	Version             string      `json:"version,omitempty"`
	Artifact            string      `json:"artifact,omitempty"`
	PromotionChangeID   *uuid.UUID  `json:"promotion_change_id,omitempty" merge:"firstSet"`
	TargetDeploymentIDs []uuid.UUID `json:"target_deployment_ids,omitempty" merge:"append"`
}

func (g PromotionGroup) Kind() string { return GroupTypePromotion }
func (g PromotionGroup) MarshalJSON() ([]byte, error) {
	type raw PromotionGroup
	return marshalWithKind(g.Kind(), raw(g))
}

type TemporaryPermissionGroup struct {
	UserID          string     `json:"user_id,omitempty"`
	RoleID          string     `json:"role_id,omitempty"`
	Scope           string     `json:"scope,omitempty"`
	GrantChangeID   *uuid.UUID `json:"grant_change_id,omitempty"  merge:"firstSet"`
	RevokeChangeID  *uuid.UUID `json:"revoke_change_id,omitempty" merge:"firstSet"`
	DurationSeconds *int64     `json:"duration_seconds,omitempty"`
}

func (g TemporaryPermissionGroup) Kind() string { return GroupTypeTemporaryPermission }
func (g TemporaryPermissionGroup) MarshalJSON() ([]byte, error) {
	type raw TemporaryPermissionGroup
	return marshalWithKind(g.Kind(), raw(g))
}

type IncidentResponseGroup struct {
	IncidentID     string      `json:"incident_id,omitempty"`
	OpenedAt       time.Time   `json:"opened_at,omitempty"  merge:"min"`
	ClosedAt       time.Time   `json:"closed_at,omitempty"  merge:"max"`
	PlaybookRunIDs []uuid.UUID `json:"playbook_run_ids,omitempty" merge:"append"`
}

func (g IncidentResponseGroup) Kind() string { return GroupTypeIncidentResponse }
func (g IncidentResponseGroup) MarshalJSON() ([]byte, error) {
	type raw IncidentResponseGroup
	return marshalWithKind(g.Kind(), raw(g))
}

type CustomGroup struct {
	Fields map[string]any `json:"fields,omitempty" merge:"mapMerge"`
}

func (g CustomGroup) Kind() string { return GroupTypeCustom }
func (g CustomGroup) MarshalJSON() ([]byte, error) {
	type raw CustomGroup
	return marshalWithKind(g.Kind(), raw(g))
}

// UnmarshalGroupDetails inspects the "kind" envelope and returns the matching
// concrete GroupType value.
func UnmarshalGroupDetails(raw json.RawMessage) (GroupType, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var envelope struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode group details envelope: %w", err)
	}

	switch envelope.Kind {
	case GroupTypeStartup:
		var g StartupGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	case GroupTypeDeployment:
		var g DeploymentGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	case GroupTypePromotion:
		var g PromotionGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	case GroupTypeTemporaryPermission:
		var g TemporaryPermissionGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	case GroupTypeIncidentResponse:
		var g IncidentResponseGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	case GroupTypeCustom:
		var g CustomGroup
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		return g, nil
	default:
		return nil, fmt.Errorf("unknown group kind %q", envelope.Kind)
	}
}
