package changegroup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// hashKey combines the rule name and raw key expression output into the
// final stored correlation_key. Including the rule name prevents accidental
// collisions between two rules whose key expressions happen to produce the
// same string.
func hashKey(ruleName, rawKey string) string {
	h := sha256.Sum256([]byte(ruleName + "|" + rawKey))
	return ruleName + ":" + hex.EncodeToString(h[:])
}

// advisoryLockKey converts a (type, correlation_key) pair into an int64
// suitable for pg_advisory_xact_lock. The lock serializes concurrent upserts
// for the same logical group across workers.
func advisoryLockKey(groupType, correlationKey string) int64 {
	h := fnv.New64a()
	h.Write([]byte(groupType))
	h.Write([]byte{0})
	h.Write([]byte(correlationKey))
	return int64(h.Sum64()) //nolint:gosec // wraparound is fine for advisory locks
}

// upsertAndAttach finds-or-creates an open group for the given correlation
// key, evaluates Details and Summary against the full member list (including
// the new change), merges the result into stored details, and sets
// change.GroupID. Runs in a single transaction with an advisory lock.
func (e *Engine) upsertAndAttach(
	ctx context.Context,
	rule *GroupingRule,
	correlationKey string,
	change *models.ConfigChange,
) error {
	return ctx.DB().Transaction(func(tx *gorm.DB) error {
		// Placeholder group type used only for the advisory lock key while we
		// look up or create the real row. We don't know the final group type
		// until we evaluate Details, so lock on (rule.Name, correlationKey) —
		// rule name is already baked into correlationKey, so locking on that
		// alone is sufficient.
		lockKey := advisoryLockKey(rule.Name, correlationKey)
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(?)`, lockKey).Error; err != nil {
			return fmt.Errorf("advisory lock: %w", err)
		}

		// Find existing open group for this correlation_key, regardless of type.
		// The unique index is (type, correlation_key) WHERE status='open', but
		// rule name + hash in the key already makes collisions across types
		// astronomically unlikely.
		var existing models.ChangeGroup
		err := tx.Where("correlation_key = ? AND status = ?", correlationKey, models.ChangeGroupStatusOpen).
			Take(&existing).Error

		var members []models.ConfigChange
		var currentGroup *models.ChangeGroup
		if err == nil {
			currentGroup = &existing
			if err := tx.Where("group_id = ?", existing.ID).
				Order("created_at ASC, id ASC").
				Find(&members).Error; err != nil {
				return fmt.Errorf("load group members: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("lookup open group: %w", err)
		}

		// Append the triggering change as the last member when computing env.
		memberMaps := make([]map[string]any, 0, len(members)+1)
		for i := range members {
			memberMaps = append(memberMaps, changeAsMap(&members[i]))
		}
		memberMaps = append(memberMaps, changeAsMap(change))

		env := Env{
			Change:  changeAsMap(change),
			Changes: memberMaps,
			Flat:    changeAsMap(change),
		}
		if currentGroup != nil {
			env.Group = groupAsMap(currentGroup)
		}

		// Details is required.
		incomingDetails, err := e.evaluator.EvalGroupDetails(rule.detailsProgram, env)
		if err != nil {
			return &EvalError{Rule: rule.Name, Field: "details", Err: err}
		}
		if incomingDetails == nil {
			return fmt.Errorf("changegroup: rule %q details evaluated to nil", rule.Name)
		}

		summary := ""
		if rule.summaryProgram != nil {
			summary, err = e.evaluator.EvalString(rule.summaryProgram, env)
			if err != nil {
				return &EvalError{Rule: rule.Name, Field: "summary", Err: err}
			}
		}

		// Determine the effective group for this attach.
		if currentGroup == nil {
			createdAt := time.Now().UTC()
			if change.CreatedAt != nil {
				createdAt = *change.CreatedAt
			}
			newGroup := models.ChangeGroup{
				ID:             uuid.New(),
				Type:           incomingDetails.Kind(),
				Summary:        summary,
				CorrelationKey: correlationKey,
				Source:         models.ChangeGroupSourceRule + ":" + rule.Name,
				RuleName:       &rule.Name,
				Status:         models.ChangeGroupStatusOpen,
				StartedAt:      createdAt,
				LastMemberAt:   createdAt,
				MemberCount:    0, // trigger will bump on attach
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt,
			}
			raw, err := json.Marshal(incomingDetails)
			if err != nil {
				return fmt.Errorf("marshal group details: %w", err)
			}
			newGroup.Details = types.JSON(raw)
			if err := tx.Create(&newGroup).Error; err != nil {
				return fmt.Errorf("create change_group: %w", err)
			}
			currentGroup = &newGroup
		} else {
			stored, err := currentGroup.TypedDetails()
			if err != nil {
				return fmt.Errorf("unmarshal stored group details: %w", err)
			}
			merged, err := Merge(stored, incomingDetails)
			if err != nil {
				return fmt.Errorf("merge group details: %w", err)
			}
			raw, err := json.Marshal(merged)
			if err != nil {
				return fmt.Errorf("marshal merged details: %w", err)
			}
			updates := map[string]any{
				"details":    types.JSON(raw),
				"updated_at": time.Now().UTC(),
			}
			if summary != "" {
				updates["summary"] = summary
			}
			if err := tx.Model(&models.ChangeGroup{}).
				Where("id = ?", currentGroup.ID).
				Updates(updates).Error; err != nil {
				return fmt.Errorf("update change_group: %w", err)
			}
		}

		// Attach the change to the group. The 047 trigger maintains
		// member_count / last_member_at / started_at.
		change.GroupID = &currentGroup.ID
		if err := tx.Model(&models.ConfigChange{}).
			Where("id = ?", change.ID).
			Update("group_id", currentGroup.ID).Error; err != nil {
			return fmt.Errorf("attach change to group: %w", err)
		}
		return nil
	})
}

// groupAsMap projects a ChangeGroup into the CEL binding shape.
func groupAsMap(g *models.ChangeGroup) map[string]any {
	return map[string]any{
		"id":              g.ID.String(),
		"type":            g.Type,
		"summary":         g.Summary,
		"correlation_key": g.CorrelationKey,
		"source":          g.Source,
		"status":          g.Status,
		"started_at":      g.StartedAt,
		"ended_at":        g.EndedAt,
		"last_member_at":  g.LastMemberAt,
		"member_count":    g.MemberCount,
		"details":         g.Details,
	}
}
