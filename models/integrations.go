package models

import (
	"time"

	"github.com/google/uuid"
)

type IntegrationType string

const (
	IntegrationTypeScraper         IntegrationType = "scrapers"
	IntegrationTypeLoggingBackends IntegrationType = "logging_backends"
	IntegrationTypeTopology        IntegrationType = "topology"
)

type Integration struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Integration  IntegrationType `json:"integration"`
	Source       string          `json:"source"`
	AgentID      *uuid.UUID      `json:"agent_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty"`
	CreatedBy    *uuid.UUID      `json:"created_by,omitempty"`
	JobName      string          `json:"job_name"`
	JobSuccess   int             `json:"job_success_count"`
	JobError     int             `json:"job_error_count"`
	JobDetails   string          `json:"job_details"`
	JobHostname  string          `json:"job_hostname"`
	JobDuration  int             `json:"job_duration_millis"`
	JobResource  string          `json:"job_resource_type"`
	JobStatus    string          `json:"job_status"`
	JobTimeStart time.Time       `json:"job_time_start"`
	JobTimeEnd   time.Time       `json:"job_time_end"`
	JobCreatedAt time.Time       `json:"job_created_at"`
}

func (i Integration) PK() string {
	return i.ID.String()
}

func (t *Integration) TableName() string {
	return "integrations_with_status"
}
