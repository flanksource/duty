package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/types"
)

// ChangeGroup represents the change_groups database table — a logical
// grouping of correlated config_changes rows.
type ChangeGroup struct {
	ID             uuid.UUID  `gorm:"primaryKey;column:id;default:generate_ulid()" json:"id"`
	Type           string     `gorm:"column:type" json:"type"`
	Summary        string     `gorm:"column:summary" json:"summary"`
	CorrelationKey string     `gorm:"column:correlation_key" json:"correlation_key"`
	Source         string     `gorm:"column:source" json:"source"`
	RuleName       *string    `gorm:"column:rule_name" json:"rule_name,omitempty"`
	Status         string     `gorm:"column:status" json:"status"`
	StartedAt      time.Time  `gorm:"column:started_at" json:"started_at"`
	EndedAt        *time.Time `gorm:"column:ended_at" json:"ended_at,omitempty"`
	LastMemberAt   time.Time  `gorm:"column:last_member_at" json:"last_member_at"`
	MemberCount    int        `gorm:"column:member_count" json:"member_count"`
	Details        types.JSON `gorm:"column:details" json:"details,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

const (
	ChangeGroupStatusOpen   = "open"
	ChangeGroupStatusClosed = "closed"

	ChangeGroupSourceExplicit = "explicit"
	ChangeGroupSourceRule     = "rule"
)

func (ChangeGroup) TableName() string { return "change_groups" }

func (g ChangeGroup) PK() string { return g.ID.String() }

// TypedDetails returns the strongly-typed GroupType for the Details column
// by inspecting the "kind" envelope.
func (g ChangeGroup) TypedDetails() (types.GroupType, error) {
	if len(g.Details) == 0 {
		return nil, nil
	}
	return types.UnmarshalGroupDetails(json.RawMessage(g.Details))
}
