package types

import (
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type PermissionGroupSubjects struct {
	Notifications []PermissionGroupSelector `json:"notifications,omitempty"`
	People        []string                  `json:"people,omitempty"`
	Teams         []string                  `json:"teams,omitempty"`
}

// +kubebuilder:object:generate=true
type PermissionGroupSelector struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

func (t PermissionGroupSelector) Empty() bool {
	return t.Name == "" && t.Namespace == ""
}

type PermissionObject struct {
	Playbooks  []types.ResourceSelector `json:"playbooks,omitempty"`
	Configs    []types.ResourceSelector `json:"configs,omitempty"`
	Components []types.ResourceSelector `json:"components,omitempty"`
}

// GlobalObject checks if the object selector semantically maps to a global object
// and returns the corresponding global object if applicable.
// For example:
//
//	configs:
//		- name: '*'
//
// is interpreted as the object: catalog.
func (t *PermissionObject) GlobalObject() (string, bool) {
	if len(t.Playbooks) == 1 && len(t.Configs) == 0 && len(t.Components) == 0 && t.Playbooks[0].Wildcard() {
		return policy.ObjectPlaybooks, true
	}

	if len(t.Configs) == 1 && len(t.Playbooks) == 0 && len(t.Components) == 0 && t.Configs[0].Wildcard() {
		return policy.ObjectCatalog, true
	}

	if len(t.Components) == 1 && len(t.Playbooks) == 0 && len(t.Configs) == 0 && t.Components[0].Wildcard() {
		return policy.ObjectTopology, true
	}

	return "", false
}

func (t PermissionObject) RequiredMatchCount() int {
	var count int
	if len(t.Playbooks) > 0 {
		count++
	}
	if len(t.Configs) > 0 {
		count++
	}
	if len(t.Components) > 0 {
		count++
	}

	return count
}
