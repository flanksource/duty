package changegroup

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// Create inserts a new change_group with an explicit source. Producers call
// this when they know upfront which changes belong together (e.g. a playbook
// emits a coordinated deployment across several configs).
//
// The correlation key defaults to the group id if empty — explicit groups are
// keyed on identity, not content.
func Create(ctx context.Context, group models.ChangeGroup) (uuid.UUID, error) {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	if group.CorrelationKey == "" {
		group.CorrelationKey = "explicit:" + group.ID.String()
	}
	if group.Source == "" {
		group.Source = models.ChangeGroupSourceExplicit
	}
	if group.Status == "" {
		group.Status = models.ChangeGroupStatusOpen
	}
	now := time.Now().UTC()
	if group.StartedAt.IsZero() {
		group.StartedAt = now
	}
	if group.LastMemberAt.IsZero() {
		group.LastMemberAt = group.StartedAt
	}
	if group.CreatedAt.IsZero() {
		group.CreatedAt = now
	}
	if group.UpdatedAt.IsZero() {
		group.UpdatedAt = now
	}

	if err := ctx.DB().Create(&group).Error; err != nil {
		return uuid.Nil, err
	}
	return group.ID, nil
}

// CreateTyped is a convenience that builds a ChangeGroup from a typed
// GroupType details value plus the usual metadata fields.
func CreateTyped(
	ctx context.Context,
	kind types.GroupType,
	summary string,
) (uuid.UUID, error) {
	raw, err := json.Marshal(kind)
	if err != nil {
		return uuid.Nil, err
	}
	return Create(ctx, models.ChangeGroup{
		Type:    kind.Kind(),
		Summary: summary,
		Details: types.JSON(raw),
	})
}

// Assign attaches the given already-persisted config_changes rows to the
// given group. The 047 trigger maintains member_count and time bounds.
func Assign(ctx context.Context, groupID uuid.UUID, changeIDs ...string) error {
	if len(changeIDs) == 0 {
		return nil
	}
	return ctx.DB().Model(&models.ConfigChange{}).
		Where("id IN ?", changeIDs).
		Update("group_id", groupID).Error
}
