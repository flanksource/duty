package models

import (
	"fmt"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type CheckHealthStatus string

const (
	CheckStatusHealthy   = "healthy"
	CheckStatusUnhealthy = "unhealthy"
)

var CheckHealthStatuses = []CheckHealthStatus{
	CheckStatusHealthy,
	CheckStatusUnhealthy,
}

type Check struct {
	ID                 uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	CanaryID           uuid.UUID           `json:"canary_id"`
	AgentID            uuid.UUID           `json:"agent_id,omitempty"`
	Spec               types.JSON          `json:"-"`
	Type               string              `json:"type"`
	Name               string              `json:"name"`
	Labels             types.JSONStringMap `json:"labels" gorm:"type:jsonstringmap"`
	Description        string              `json:"description,omitempty"`
	Status             CheckHealthStatus   `json:"status,omitempty"`
	Owner              string              `json:"owner,omitempty"`
	Severity           string              `json:"severity,omitempty"`
	Icon               string              `json:"icon,omitempty"`
	Transformed        bool                `json:"transformed,omitempty"`
	LastRuntime        *time.Time          `json:"last_runtime,omitempty"`
	NextRuntime        *time.Time          `json:"next_runtime,omitempty"`
	LastTransitionTime *time.Time          `json:"last_transition_time,omitempty"`
	CreatedAt          *time.Time          `json:"created_at,omitempty"`
	UpdatedAt          *time.Time          `json:"updated_at,omitempty"`
	DeletedAt          *time.Time          `json:"deleted_at,omitempty"`
	SilencedAt         *time.Time          `json:"silenced_at,omitempty"`

	// Auxiliary fields
	CanaryName   string        `json:"canary_name" gorm:"-"`
	Namespace    string        `json:"namespace"  gorm:"-"`     // Namespace of the parent canary
	ComponentIDs []string      `json:"component_ids"  gorm:"-"` // Linked component ids
	Uptime       Uptime        `json:"uptime"  gorm:"-"`
	Latency      Latency       `json:"latency"  gorm:"-"`
	Statuses     []CheckStatus `json:"checkStatuses"  gorm:"-"`
	DisplayType  string        `json:"display_type,omitempty"  gorm:"-"`
}

func (c Check) ToString() string {
	return fmt.Sprintf("%s-%s-%s", c.Name, c.Type, c.Description)
}

func (c Check) GetDescription() string {
	return c.Description
}

type Checks []*Check

func (c Checks) Len() int {
	return len(c)
}

func (c Checks) Less(i, j int) bool {
	return c[i].ToString() < c[j].ToString()
}

func (c Checks) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Checks) Find(key string) *Check {
	for _, check := range c {
		if check.Name == key {
			return check
		}
	}
	return nil
}

type Uptime struct {
	Passed   int        `json:"passed"`
	Failed   int        `json:"failed"`
	P100     float64    `json:"p100,omitempty"`
	LastPass *time.Time `json:"last_pass,omitempty"`
	LastFail *time.Time `json:"last_fail,omitempty"`
}

func (u Uptime) String() string {
	if u.Passed == 0 && u.Failed == 0 {
		return ""
	}
	if u.Passed == 0 {
		return fmt.Sprintf("0/%d 0%%", u.Failed)
	}
	percentage := 100.0 * (1 - (float64(u.Failed) / float64(u.Passed+u.Failed)))
	return fmt.Sprintf("%d/%d (%0.1f%%)", u.Passed, u.Passed+u.Failed, percentage)
}

type CheckStatus struct {
	CheckID   uuid.UUID `json:"check_id"`
	Status    bool      `json:"status"`
	Invalid   bool      `json:"invalid,omitempty"`
	Time      string    `json:"time"`
	Duration  int       `json:"duration"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Detail    any       `json:"-" gorm:"-"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

func (s CheckStatus) GetTime() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s.Time)
}

func (CheckStatus) TableName() string {
	return "check_statuses"
}

// CheckStatusAggregate1h represents the `check_statuses_1h` table
type CheckStatusAggregate1h struct {
	CheckID   string    `gorm:"column:check_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	Duration  int       `gorm:"column:duration"`
	Total     int       `gorm:"column:total"`
	Passed    int       `gorm:"column:passed"`
	Failed    int       `gorm:"column:failed"`
}

func (CheckStatusAggregate1h) TableName() string {
	return "check_statuses_1h"
}

// CheckStatusAggregate1d represents the `check_statuses_1d` table
type CheckStatusAggregate1d struct {
	CheckID   string    `gorm:"column:check_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	Duration  int       `gorm:"column:duration"`
	Total     int       `gorm:"column:total"`
	Passed    int       `gorm:"column:passed"`
	Failed    int       `gorm:"column:failed"`
}

func (CheckStatusAggregate1d) TableName() string {
	return "check_statuses_1d"
}
