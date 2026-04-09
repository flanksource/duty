package changegroup

import (
	"sort"
	"sync"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
)

// Engine evaluates GroupingRules against incoming config_changes and keeps
// change_groups in sync. It is safe for concurrent use once rules are loaded.
type Engine struct {
	mu        sync.RWMutex
	rules     []*GroupingRule
	evaluator Evaluator
}

// New returns an Engine pre-loaded with the given rules and evaluator.
// The caller is responsible for calling Validate on each rule before passing
// it in — New re-validates and returns an error if any rule fails.
func New(evaluator Evaluator, rules []GroupingRule) (*Engine, error) {
	e := &Engine{evaluator: evaluator}
	if err := e.SetRules(rules); err != nil {
		return nil, err
	}
	return e, nil
}

// SetRules replaces the engine's rule set atomically. All rules are
// (re-)validated; if any rule fails, the previous rule set is preserved and
// the error is returned.
func (e *Engine) SetRules(rules []GroupingRule) error {
	if e.evaluator == nil {
		// Allow rule loading with no evaluator only if every rule has empty
		// CEL expressions (currently never true). Require evaluator.
		return ErrMissingEvaluator
	}

	compiled := make([]*GroupingRule, 0, len(rules))
	for i := range rules {
		r := rules[i]
		if err := r.Validate(e.evaluator); err != nil {
			return err
		}
		compiled = append(compiled, &r)
	}
	sort.SliceStable(compiled, func(i, j int) bool {
		return compiled[i].Priority > compiled[j].Priority
	})

	e.mu.Lock()
	e.rules = compiled
	e.mu.Unlock()
	return nil
}

// Rules returns a snapshot of the currently loaded rules for inspection/tests.
func (e *Engine) Rules() []*GroupingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*GroupingRule, len(e.rules))
	copy(out, e.rules)
	return out
}

// Evaluate runs the rule engine against a single already-persisted
// config_changes row. Matching rules will create or update a change_group and
// set change.GroupID. If the change already has a GroupID, Evaluate is a no-op
// (producers are trusted).
func (e *Engine) Evaluate(ctx context.Context, change *models.ConfigChange) error {
	if change == nil {
		return nil
	}
	if change.GroupID != nil {
		return nil // explicit path — respect producer assignment
	}

	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	for _, rule := range rules {
		matched, err := e.tryRule(ctx, rule, change)
		if err != nil {
			return err
		}
		if matched {
			return nil
		}
	}
	return nil
}

// tryRule attempts to apply a single rule to the change. Returns (true, nil)
// on a successful attach, (false, nil) on a non-match.
func (e *Engine) tryRule(ctx context.Context, rule *GroupingRule, change *models.ConfigChange) (bool, error) {
	if !rule.Matches(change.ChangeType) {
		return false, nil
	}

	env := buildSingleChangeEnv(change)

	if rule.filterProgram != nil {
		ok, err := e.evaluator.EvalBool(rule.filterProgram, env)
		if err != nil {
			return false, &EvalError{Rule: rule.Name, Field: "filter", Err: err}
		}
		if !ok {
			return false, nil
		}
	}

	rawKey, err := e.evaluator.EvalString(rule.keyProgram, env)
	if err != nil {
		return false, &EvalError{Rule: rule.Name, Field: "key", Err: err}
	}
	if rawKey == "" {
		return false, nil
	}
	correlationKey := hashKey(rule.Name, rawKey)

	if err := e.upsertAndAttach(ctx, rule, correlationKey, change); err != nil {
		return false, err
	}
	return true, nil
}

// buildSingleChangeEnv creates an Env whose Changes contains only the
// triggering change (plus the flat shortcuts). Used on the first call; the
// upsert path rebuilds the env with all persisted members before re-running
// Details / Summary.
func buildSingleChangeEnv(change *models.ConfigChange) Env {
	m := changeAsMap(change)
	return Env{
		Change:  m,
		Changes: []map[string]any{m},
		Flat:    m,
	}
}

// changeAsMap projects a ConfigChange into the CEL binding shape.
func changeAsMap(c *models.ConfigChange) map[string]any {
	var groupID any
	if c.GroupID != nil {
		groupID = c.GroupID.String()
	}
	return map[string]any{
		"id":                  c.ID,
		"external_id":         c.ExternalID,
		"external_change_id":  derefString(c.ExternalChangeID),
		"config_id":           c.ConfigID,
		"change_type":         c.ChangeType,
		"severity":            string(c.Severity),
		"source":              c.Source,
		"summary":             c.Summary,
		"patches":             c.Patches,
		"diff":                c.Diff,
		"fingerprint":         c.Fingerprint,
		"details":             c.Details,
		"created_at":          c.CreatedAt,
		"created_by":          c.CreatedBy,
		"external_created_by": derefString(c.ExternalCreatedBy),
		"count":               c.Count,
		"group_id":            groupID,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

