package models

import (
	"maps"
	"slices"

	"github.com/flanksource/duty/types"
)

func (check Check) Clone() Check {
	clone := check
	clone.Spec = slices.Clone(check.Spec)
	clone.Labels = maps.Clone(check.Labels)
	clone.LastTransitionTime = clonePtr(check.LastTransitionTime)
	clone.CreatedAt = clonePtr(check.CreatedAt)
	clone.UpdatedAt = clonePtr(check.UpdatedAt)
	clone.DeletedAt = clonePtr(check.DeletedAt)
	clone.SilencedAt = clonePtr(check.SilencedAt)
	clone.ComponentIDs = slices.Clone(check.ComponentIDs)
	clone.Uptime = cloneUptime(check.Uptime)
	clone.Statuses = cloneCheckStatuses(check.Statuses)
	clone.EarliestRuntime = clonePtr(check.EarliestRuntime)
	clone.LatestRuntime = clonePtr(check.LatestRuntime)
	return clone
}

func (config ConfigItem) Clone() ConfigItem {
	clone := config
	clone.ScraperID = clonePtr(config.ScraperID)
	clone.ExternalID = slices.Clone(config.ExternalID)
	clone.Type = clonePtr(config.Type)
	clone.Status = clonePtr(config.Status)
	clone.Health = clonePtr(config.Health)
	clone.Name = clonePtr(config.Name)
	clone.Description = clonePtr(config.Description)
	clone.Config = clonePtr(config.Config)
	clone.Source = clonePtr(config.Source)
	clone.ParentID = clonePtr(config.ParentID)
	clone.Labels = cloneJSONStringMapPtr(config.Labels)
	clone.Tags = maps.Clone(config.Tags)
	clone.Properties = cloneTypePropertiesPtr(config.Properties)
	clone.UpdatedAt = clonePtr(config.UpdatedAt)
	clone.DeletedAt = clonePtr(config.DeletedAt)
	clone.configJson = cloneAnyMap(config.configJson)
	return clone
}

func (component Component) Clone() Component {
	clone := component
	clone.TopologyID = clonePtr(component.TopologyID)
	clone.ParentId = clonePtr(component.ParentId)
	clone.Labels = maps.Clone(component.Labels)
	clone.Health = clonePtr(component.Health)
	clone.LogSelectors = cloneLogSelectors(component.LogSelectors)
	clone.Selectors = cloneResourceSelectors(component.Selectors)
	clone.Configs = cloneConfigQueries(component.Configs)
	clone.ComponentChecks = cloneComponentChecks(component.ComponentChecks)
	clone.Properties = cloneModelProperties(component.Properties)
	clone.Summary = cloneSummary(component.Summary)
	clone.CreatedBy = clonePtr(component.CreatedBy)
	clone.UpdatedAt = clonePtr(component.UpdatedAt)
	clone.DeletedAt = clonePtr(component.DeletedAt)
	clone.ConfigID = clonePtr(component.ConfigID)
	clone.Checks = maps.Clone(component.Checks)
	clone.Incidents = cloneNestedIntMap(component.Incidents)
	clone.Analysis = cloneNestedIntMap(component.Analysis)
	clone.Components = cloneComponents(component.Components)
	clone.RelationshipID = clonePtr(component.RelationshipID)
	clone.Children = slices.Clone(component.Children)
	clone.Parents = slices.Clone(component.Parents)
	return clone
}

func clonePtr[T any](in *T) *T {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneJSONStringMapPtr(in *types.JSONStringMap) *types.JSONStringMap {
	if in == nil {
		return nil
	}
	out := maps.Clone(*in)
	return &out
}

func cloneUptime(in types.Uptime) types.Uptime {
	out := in
	out.P100 = clonePtr(in.P100)
	out.LastPass = clonePtr(in.LastPass)
	out.LastFail = clonePtr(in.LastFail)
	return out
}

func cloneCheckStatuses(in []CheckStatus) []CheckStatus {
	out := slices.Clone(in)
	for i := range out {
		out[i].Detail = cloneAny(in[i].Detail)
	}
	return out
}

func cloneComponents(in Components) Components {
	if in == nil {
		return nil
	}
	out := make(Components, len(in))
	for i, component := range in {
		if component != nil {
			clone := component.Clone()
			out[i] = &clone
		}
	}
	return out
}

func cloneModelProperties(in Properties) Properties {
	if in == nil {
		return nil
	}
	out := make(Properties, len(in))
	for i, property := range in {
		out[i] = property.DeepCopy()
	}
	return out
}

func cloneTypePropertiesPtr(in *types.Properties) *types.Properties {
	if in == nil {
		return nil
	}
	out := cloneTypeProperties(*in)
	return &out
}

func cloneTypeProperties(in types.Properties) types.Properties {
	if in == nil {
		return nil
	}
	out := make(types.Properties, len(in))
	for i, property := range in {
		out[i] = property.DeepCopy()
	}
	return out
}

func cloneLogSelectors(in types.LogSelectors) types.LogSelectors {
	if in == nil {
		return nil
	}
	out := make(types.LogSelectors, len(in))
	for i, selector := range in {
		out[i] = *selector.DeepCopy()
	}
	return out
}

func cloneResourceSelectors(in types.ResourceSelectors) types.ResourceSelectors {
	if in == nil {
		return nil
	}
	out := make(types.ResourceSelectors, len(in))
	for i, selector := range in {
		out[i] = cloneResourceSelector(selector)
	}
	return out
}

func cloneResourceSelector(in types.ResourceSelector) types.ResourceSelector {
	return *in.DeepCopy()
}

func cloneConfigQueries(in types.ConfigQueries) types.ConfigQueries {
	if in == nil {
		return nil
	}
	out := make(types.ConfigQueries, len(in))
	for i, query := range in {
		out[i] = query.DeepCopy()
	}
	return out
}

func cloneComponentChecks(in types.ComponentChecks) types.ComponentChecks {
	if in == nil {
		return nil
	}
	out := make(types.ComponentChecks, len(in))
	for i, check := range in {
		out[i] = *check.DeepCopy()
	}
	return out
}

func cloneSummary(in types.Summary) types.Summary {
	return *in.DeepCopy()
}

func cloneNestedIntMap(in map[string]map[string]int) map[string]map[string]int {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string]int, len(in))
	for key, val := range in {
		out[key] = maps.Clone(val)
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, val := range in {
		out[key] = cloneAny(val)
	}
	return out
}

func cloneAny(in any) any {
	switch val := in.(type) {
	case map[string]any:
		return cloneAnyMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = cloneAny(item)
		}
		return out
	case map[string]string:
		return maps.Clone(val)
	case []string:
		return slices.Clone(val)
	case types.JSONStringMap:
		return maps.Clone(val)
	case types.JSON:
		return slices.Clone(val)
	default:
		return val
	}
}
