package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
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

func (i Incident) TableName() string {
	return "incidents"
}

func (i Incident) PK() string {
	return i.ID.String()
}

func DeleteAllIncidents(db *gorm.DB, incidents ...Incident) error {
	ids := lo.Map(incidents, func(i Incident, _ int) string {
		return i.ID.String()
	})

	if err := db.Exec(`DELETE FROM incident_histories where incident_id in (?)`, ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM evidences where hypothesis_id in (select id from hypotheses where incident_id in (?) )", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM hypotheses where incident_id in (?)", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM comments where incident_id in (?)", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM incident_relationships where incident_id in (?) or related_id in (?)", ids, ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM comment_responders where responder_id in (select id from responders where incident_id in (?))", ids).Error; err != nil {
		return err
	}

	if err := db.Exec("DELETE FROM responders where incident_id in (?)", ids).Error; err != nil {
		return err
	}

	if err := db.Exec("DELETE FROM incidents where id in (?)", ids).Error; err != nil {
		return err
	}
	return nil
}

func (i Incident) AsMap(removeFields ...string) map[string]any {
	return asMap(i, removeFields...)
}
