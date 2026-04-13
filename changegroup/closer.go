package changegroup

import (
	"encoding/json"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// CloseStaleGroups closes open change_groups whose last_member_at is older
// than the effective close window for the group's rule. Called periodically
// by StartCloser.
//
// For TemporaryPermissionGroup specifically, this also computes
// DurationSeconds from started_at/ended_at at close time.
func (e *Engine) CloseStaleGroups(ctx context.Context, now time.Time) (int, error) {
	// Build the rule → CloseAfter lookup snapshot.
	e.mu.RLock()
	closeAfterByRule := make(map[string]time.Duration, len(e.rules))
	for _, r := range e.rules {
		ca := r.CloseAfter.Std()
		if ca == 0 {
			// 0 means "never time out" — skip timeout-close for this rule.
			continue
		}
		closeAfterByRule[r.Name] = ca
	}
	e.mu.RUnlock()

	if len(closeAfterByRule) == 0 {
		return 0, nil
	}

	var candidates []models.ChangeGroup
	if err := ctx.DB().
		Where("status = ?", models.ChangeGroupStatusOpen).
		Find(&candidates).Error; err != nil {
		return 0, err
	}

	closed := 0
	for i := range candidates {
		g := &candidates[i]
		if g.RuleName == nil {
			continue
		}
		window, ok := closeAfterByRule[*g.RuleName]
		if !ok {
			continue
		}
		if now.Sub(g.LastMemberAt) < window {
			continue
		}
		if err := finalizeClose(ctx, g, g.LastMemberAt); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

// finalizeClose writes the terminal state for a group: status=closed,
// ended_at set, and — for TemporaryPermissionGroup — DurationSeconds computed.
func finalizeClose(ctx context.Context, g *models.ChangeGroup, endedAt time.Time) error {
	updates := map[string]any{
		"status":     models.ChangeGroupStatusClosed,
		"ended_at":   endedAt,
		"updated_at": time.Now().UTC(),
	}

	stored, err := g.TypedDetails()
	if err == nil && stored != nil {
		if tp, ok := stored.(types.TemporaryPermissionGroup); ok {
			dur := int64(endedAt.Sub(g.StartedAt).Seconds())
			tp.DurationSeconds = &dur
			raw, err := json.Marshal(tp)
			if err == nil {
				updates["details"] = types.JSON(raw)
			}
		}
	}

	return ctx.DB().Model(&models.ChangeGroup{}).
		Where("id = ?", g.ID).
		Updates(updates).Error
}

// StartCloser runs CloseStaleGroups on a ticker until ctx is cancelled.
// Intended to be spawned once per process by the job scheduler.
func (e *Engine) StartCloser(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				_, _ = e.CloseStaleGroups(ctx, now)
			}
		}
	}()
}
