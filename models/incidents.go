package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type IncidentType string

var (
	IncidentTypeAvailability  IncidentType = "availability"
	IncidentTypeCost          IncidentType = "cost"
	IncidentTypePerformance   IncidentType = "performance"
	IncidentTypeSecurity      IncidentType = "security"
	IncidentTypeTechnicalDebt IncidentType = "technical_debt"
	IncidentTypeCompliance    IncidentType = "compliance"
	IncidentTypeIntegration   IncidentType = "integration"
)

type IncidentStatus string

var (
	IncidentStatusOpen      IncidentStatus = "open"
	IncidentStatusClosed    IncidentStatus = "closed"
	IncidentStatusMitigated IncidentStatus = "mitigated"
	IncidentStatusResolved  IncidentStatus = "resolved"
	IncidentStatusCancelled IncidentStatus = "cancelled"
)

type Incident struct {
	ID             uuid.UUID      `json:"id,omitempty" gorm:"default:generate_ulid()"`
	IncidentID     string         `json:"incident_id,omitempty" gorm:"default:format_incident_id(NEXTVAL('incident_id_sequence'))"`
	Title          string         `json:"title,omitempty"`
	Description    string         `json:"description,omitempty"`
	Type           IncidentType   `json:"type,omitempty"`
	Status         IncidentStatus `json:"status,omitempty"`
	Severity       Severity       `json:"severity,omitempty"`
	CreatedAt      *time.Time     `json:"created_at,omitempty"`
	UpdatedAt      *time.Time     `json:"updated_at,omitempty"`
	Acknowledged   *time.Time     `json:"acknowledged,omitempty"`
	Resolved       *time.Time     `json:"resolved,omitempty"`
	Closed         *time.Time     `json:"closed,omitempty"`
	CreatedBy      uuid.UUID      `json:"created_by,omitempty"`
	IncidentRuleID *uuid.UUID     `json:"incident_rule_id,omitempty"`
	CommanderID    *uuid.UUID     `json:"commander_id,omitempty"`
	CommunicatorID *uuid.UUID     `json:"communicator_id,omitempty"`
}

func (i Incident) AsMap() map[string]any {
	m := make(map[string]any)
	b, _ := json.Marshal(&i)
	_ = json.Unmarshal(b, &m)
	return m
}
