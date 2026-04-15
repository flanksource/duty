package query

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// ChangeGroupsSearchRequest describes filters for FindChangeGroups.
type ChangeGroupsSearchRequest struct {
	// Type is a comma-separated list of change_group.type values.
	Type string `query:"type" json:"type,omitempty"`
	// Status is open|closed. Empty means any.
	Status string `query:"status" json:"status,omitempty"`
	// ConfigID restricts to groups that have at least one member belonging to this config_id.
	ConfigID *uuid.UUID `query:"config_id" json:"config_id,omitempty"`
	// Summary is a case-insensitive substring search on the summary column.
	Summary string `query:"summary" json:"summary,omitempty"`
	// Since / Until bound started_at.
	Since *time.Time `query:"since" json:"since,omitempty"`
	Until *time.Time `query:"until" json:"until,omitempty"`
	// Pagination.
	Page     int `query:"page" json:"page,omitempty"`
	PageSize int `query:"page_size" json:"page_size,omitempty"`
}

func (r *ChangeGroupsSearchRequest) setDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize <= 0 || r.PageSize > 500 {
		r.PageSize = 50
	}
}

// FindChangeGroups returns change_groups matching the search request.
func FindChangeGroups(ctx context.Context, req ChangeGroupsSearchRequest) ([]models.ChangeGroup, error) {
	req.setDefaults()

	q := ctx.DB().Model(&models.ChangeGroup{})

	if req.Type != "" {
		q = q.Where("type IN ?", strings.Split(req.Type, ","))
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Summary != "" {
		q = q.Where("summary ILIKE ?", "%"+req.Summary+"%")
	}
	if req.Since != nil {
		q = q.Where("started_at >= ?", *req.Since)
	}
	if req.Until != nil {
		q = q.Where("started_at <= ?", *req.Until)
	}
	if req.ConfigID != nil {
		q = q.Where(
			"id IN (SELECT group_id FROM config_changes WHERE config_id = ? AND group_id IS NOT NULL)",
			*req.ConfigID,
		)
	}

	q = q.Order("started_at DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize)

	var groups []models.ChangeGroup
	if err := q.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// GetChangeGroup loads a single change_group by id.
func GetChangeGroup(ctx context.Context, id uuid.UUID) (*models.ChangeGroup, error) {
	var g models.ChangeGroup
	if err := ctx.DB().Where("id = ?", id).Take(&g).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

// GetGroupMembers returns the config_changes rows belonging to the given group.
func GetGroupMembers(ctx context.Context, id uuid.UUID) ([]models.ConfigChange, error) {
	var members []models.ConfigChange
	if err := ctx.DB().
		Where("group_id = ?", id).
		Order("created_at ASC, id ASC").
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// ChangeGroupSummary is a row from the change_groups_summary view.
type ChangeGroupSummary struct {
	ID                  uuid.UUID  `gorm:"column:id" json:"id"`
	Type                string     `gorm:"column:type" json:"type"`
	Summary             string     `gorm:"column:summary" json:"summary"`
	Source              string     `gorm:"column:source" json:"source"`
	RuleName            *string    `gorm:"column:rule_name" json:"rule_name,omitempty"`
	Status              string     `gorm:"column:status" json:"status"`
	StartedAt           time.Time  `gorm:"column:started_at" json:"started_at"`
	EndedAt             *time.Time `gorm:"column:ended_at" json:"ended_at,omitempty"`
	LastMemberAt        time.Time  `gorm:"column:last_member_at" json:"last_member_at"`
	MemberCount         int        `gorm:"column:member_count" json:"member_count"`
	DistinctConfigCount int        `gorm:"column:distinct_config_count" json:"distinct_config_count"`
	DurationSeconds     float64    `gorm:"column:duration_seconds" json:"duration_seconds"`
}

func (ChangeGroupSummary) TableName() string { return "change_groups_summary" }

// FindChangeGroupsSummary returns aggregated rows from change_groups_summary,
// optionally filtered. Filters match FindChangeGroups where applicable.
func FindChangeGroupsSummary(ctx context.Context, req ChangeGroupsSearchRequest) ([]ChangeGroupSummary, error) {
	req.setDefaults()

	q := ctx.DB().Table("change_groups_summary")

	if req.Type != "" {
		q = q.Where("type IN ?", strings.Split(req.Type, ","))
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Summary != "" {
		q = q.Where("summary ILIKE ?", "%"+req.Summary+"%")
	}
	if req.Since != nil {
		q = q.Where("started_at >= ?", *req.Since)
	}
	if req.Until != nil {
		q = q.Where("started_at <= ?", *req.Until)
	}

	q = q.Order("started_at DESC").
		Limit(req.PageSize).
		Offset((req.Page - 1) * req.PageSize)

	var out []ChangeGroupSummary
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
