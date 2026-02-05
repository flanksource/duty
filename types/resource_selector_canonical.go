package types

import (
	"strings"

	"github.com/flanksource/duty/pkg/kube/labels"
	"k8s.io/apimachinery/pkg/selection"
)

const wildcardValue = "*"
const wildcardSentinel = "__duty_wildcard__"

type selectorWildcardBehavior int

const (
	selectorWildcardIgnore selectorWildcardBehavior = iota
	selectorWildcardExists
)

// Canonical normalizes a resource selector by handling wildcard values.
func (rs ResourceSelector) Canonical() ResourceSelector {
	out := rs

	if isWildcardValue(out.ID) {
		out.ID = ""
	}

	if isWildcardValue(out.Namespace) {
		out.Namespace = ""
	}

	if isWildcardValue(out.Scope) {
		out.Scope = ""
	}

	if isWildcardValue(out.Agent) {
		out.Agent = "all"
	}

	out.Types = filterWildcardItems(out.Types)
	out.Statuses = filterWildcardItems(out.Statuses)
	out.Health = MatchExpression(filterWildcardCSV(string(out.Health)))

	out.TagSelector = canonicalizeSelector(out.TagSelector, selectorWildcardExists)
	out.LabelSelector = canonicalizeSelector(out.LabelSelector, selectorWildcardExists)
	out.FieldSelector = canonicalizeSelector(out.FieldSelector, selectorWildcardIgnore)

	return out
}

func canonicalizeSelector(selector string, behavior selectorWildcardBehavior) string {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" || trimmed == wildcardValue {
		return ""
	}

	normalized := strings.ReplaceAll(selector, wildcardValue, wildcardSentinel)
	parsed, err := labels.Parse(normalized)
	if err != nil {
		return selector
	}

	requirements, selectable := parsed.Requirements()
	if !selectable {
		return ""
	}

	var canonical []labels.Requirement
	for _, requirement := range requirements {
		if requirementHasWildcard(requirement) {
			if behavior == selectorWildcardExists && wildcardSupportsExists(requirement.Operator()) {
				converted, err := labels.NewRequirement(requirement.Key(), selection.Exists, nil)
				if err != nil {
					return selector
				}
				canonical = append(canonical, *converted)
			}
			continue
		}

		canonical = append(canonical, requirement)
	}

	if len(canonical) == 0 {
		return ""
	}

	return labels.NewSelector().Add(canonical...).String()
}

func requirementHasWildcard(requirement labels.Requirement) bool {
	for value := range requirement.Values() {
		if strings.Contains(value, wildcardSentinel) {
			return true
		}
	}
	return false
}

func wildcardSupportsExists(op selection.Operator) bool {
	return op == selection.Equals || op == selection.DoubleEquals || op == selection.In
}

func filterWildcardItems(items Items) Items {
	if len(items) == 0 {
		return nil
	}

	filtered := make(Items, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == wildcardValue || strings.TrimPrefix(trimmed, "!") == wildcardValue {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func filterWildcardCSV(value string) string {
	if value == "" {
		return ""
	}

	parts := strings.Split(value, ",")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == wildcardValue {
			continue
		}
		filtered = append(filtered, part)
	}

	return strings.Join(filtered, ",")
}

func isWildcardValue(value string) bool {
	return strings.TrimSpace(value) == wildcardValue
}
