package changegroup

import (
	"fmt"
	"sort"

	"github.com/flanksource/duty/types"
)

// Pseudo change-type identifiers used in rule DSL. Entries in
// GroupingRule.ChangeTypes starting with "@" are expanded via ExpandPseudo.
const (
	PseudoCreated   = "@created"
	PseudoChanged   = "@changed"
	PseudoDeleted   = "@deleted"
	PseudoHealthy   = "@healthy"
	PseudoUnhealthy = "@unhealthy"
	PseudoPlaybook  = "@playbook"
)

// pseudoMap defines the literal change_type set for each pseudo identifier.
// Keyed on the pseudo string (leading "@" preserved). Kept private to this
// file; callers use ExpandPseudo.
var pseudoMap = map[string]map[string]struct{}{
	PseudoCreated: setOf(
		types.ChangeTypeCreate,
		"Created",
		types.ChangeTypeUserCreated,
		types.ChangeTypeRegisterNode,
		types.ChangeTypeRunInstances,
	),
	PseudoDeleted: setOf(
		types.ChangeTypeDelete,
		"Deleted",
		types.ChangeTypeUserDeleted,
	),
	PseudoHealthy: setOf(
		"Healthy",
		types.ChangeTypeBackupCompleted,
		types.ChangeTypeBackupRestored,
		types.ChangeTypePipelineRunCompleted,
		types.ChangeTypePlaybookCompleted,
		types.ChangeTypeCertificateRenewed,
	),
	PseudoUnhealthy: setOf(
		"Unhealthy",
		types.ChangeTypeBackupFailed,
		types.ChangeTypePipelineRunFailed,
		types.ChangeTypePlaybookFailed,
		types.ChangeTypeCertificateExpired,
	),
	PseudoChanged: setOf(
		types.ChangeTypeUpdate,
		types.ChangeTypeDiff,
		types.ChangeTypeDeployment,
		types.ChangeTypePromotion,
		types.ChangeTypeRollback,
		types.ChangeTypeScaling,
		types.ChangeTypeCostChange,
		types.ChangeTypePermissionAdded,
		types.ChangeTypePermissionRemoved,
		types.ChangeTypeGroupMemberAdded,
		types.ChangeTypeGroupMemberRemoved,
	),
	PseudoPlaybook: setOf(
		types.ChangeTypePlaybookStarted,
		types.ChangeTypePlaybookCompleted,
		types.ChangeTypePlaybookFailed,
	),
}

func setOf(items ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, it := range items {
		m[it] = struct{}{}
	}
	return m
}

// ExpandPseudo returns the sorted list of literal change_type strings that
// the given pseudo identifier covers. Returns an error for unknown pseudo
// identifiers.
func ExpandPseudo(pseudo string) ([]string, error) {
	set, ok := pseudoMap[pseudo]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownPseudo, pseudo)
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// expandChangeTypes builds the deduped literal change_type set for a rule's
// ChangeTypes slice. Entries starting with "@" expand via ExpandPseudo;
// everything else is taken as a literal.
func expandChangeTypes(entries []string) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e == "" {
			continue
		}
		if e[0] == '@' {
			lits, err := ExpandPseudo(e)
			if err != nil {
				return nil, err
			}
			for _, l := range lits {
				out[l] = struct{}{}
			}
			continue
		}
		out[e] = struct{}{}
	}
	return out, nil
}
