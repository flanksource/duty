package duty

import (
	"fmt"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

// Lookup offers different ways to specify a lookup value
//
// +kubebuilder:object:generate=true
type Lookup struct {
	Expr  string `json:"expr,omitempty"`
	Value string `json:"value,omitempty"`
	Label string `json:"label,omitempty"`
}

func (t *Lookup) Eval(labels map[string]string, envVar map[string]any) (string, error) {
	if t.Value != "" {
		return t.Value, nil
	}

	if t.Label != "" {
		return labels[t.Label], nil
	}

	if t.Expr != "" {
		res, err := gomplate.RunTemplate(envVar, gomplate.Template{Expression: t.Expr})
		if err != nil {
			return "", err
		}

		return res, nil
	}

	return "", nil
}

func (t Lookup) IsEmpty() bool {
	return t.Value == "" && t.Label == "" && t.Expr == ""
}

// +kubebuilder:object:generate=true
// RelationshipSelector is the evaluated output of RelationshipSelectorTemplate.
type RelationshipSelector struct {
	ID         string            `json:"id,omitempty"`
	ExternalID string            `json:"external_id,omitempty"`
	Name       string            `json:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
	Type       string            `json:"type,omitempty"`
	Agent      string            `json:"agent,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`

	// Scope is the id parent of the resource to select.
	// Example: For config items, the scope is the scraper id
	// - for checks, it's canaries and
	// - for components, it's topology.
	Scope string `json:"scope,omitempty"`
}

func (t *RelationshipSelector) IsEmpty() bool {
	return t.ID == "" && t.ExternalID == "" && t.Name == "" && t.Namespace == "" && t.Type == "" && t.Agent == "" && len(t.Labels) == 0 && t.Scope == ""
}

func (t *RelationshipSelector) ToResourceSelector() types.ResourceSelector {
	var labelSelector string
	for k, v := range t.Labels {
		labelSelector += fmt.Sprintf("%s=%s,", k, v)
	}
	labelSelector = strings.TrimSuffix(labelSelector, ",")

	rs := types.ResourceSelector{
		ID:            t.ID,
		Scope:         t.Scope,
		Name:          t.Name,
		Agent:         t.Agent,
		LabelSelector: labelSelector,
		Namespace:     t.Namespace,
	}
	if t.Type != "" {
		rs.Types = []string{t.Type}
	}
	if t.ExternalID != "" {
		rs.FieldSelector = fmt.Sprintf("external_id=%s", t.ExternalID)
	}

	return rs
}

// +kubebuilder:object:generate=true
type RelationshipSelectorTemplate struct {
	ID         Lookup `json:"id,omitempty"`
	ExternalID Lookup `json:"external_id,omitempty"`
	Name       Lookup `json:"name,omitempty"`
	Namespace  Lookup `json:"namespace,omitempty"`
	Type       Lookup `json:"type,omitempty"`

	// Agent can be one of
	//  - agent id
	//  - agent name
	//  - 'self' (no agent)
	Agent Lookup `json:"agent,omitempty"`

	// Scope is the id of the parent of the resource to select.
	// Example: For config items, the scope is the scraper id
	// - for checks, it's canaries and
	// - for components, it's topology.
	// If left empty, the scope is the requester's scope.
	// Use `all` to disregard scope.
	Scope Lookup `json:"scope,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

func (t *RelationshipSelectorTemplate) IsEmpty() bool {
	return t.ID.IsEmpty() && t.ExternalID.IsEmpty() && t.Name.IsEmpty() && t.Namespace.IsEmpty() &&
		t.Scope.IsEmpty() && t.Type.IsEmpty() && t.Agent.IsEmpty() && len(t.Labels) == 0
}

// Eval evaluates the template and returns a RelationshipSelector.
// If any of the filter returns an empty value, the evaluation results to a nil selector.
// i.e. if a lookup is non-empty, it must return a non-empty value.
func (t *RelationshipSelectorTemplate) Eval(labels map[string]string, env map[string]any) (*RelationshipSelector, error) {
	if t.IsEmpty() {
		return nil, nil
	}

	var err error
	var output = RelationshipSelector{
		Labels: t.Labels,
	}

	if !t.ID.IsEmpty() {
		if output.ID, err = t.ID.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate id: %v for relationship: %w", t.ID, err)
		} else if output.ID == "" {
			return nil, nil
		}
	}

	if !t.ExternalID.IsEmpty() {
		if output.ExternalID, err = t.ExternalID.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate external id: %v for relationship: %w", t.ExternalID, err)
		} else if output.ExternalID == "" {
			return nil, nil
		}
	}

	if !t.Name.IsEmpty() {
		if output.Name, err = t.Name.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate name: %v for relationship: %w", t.Name, err)
		} else if output.Name == "" {
			return nil, nil
		}
	}

	if !t.Namespace.IsEmpty() {
		if output.Namespace, err = t.Namespace.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate namespace: %v for relationship: %w", t.Namespace, err)
		} else if output.Namespace == "" {
			return nil, nil
		}
	}

	if !t.Type.IsEmpty() {
		if output.Type, err = t.Type.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate type: %v for relationship: %w", t.Type, err)
		} else if output.Type == "" {
			return nil, nil
		}
	}

	if !t.Agent.IsEmpty() {
		if output.Agent, err = t.Agent.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate agent_id: %v for relationship: %w", t.Agent, err)
		} else if output.Agent == "" {
			return nil, nil
		}
	}

	if !t.Scope.IsEmpty() {
		if output.Scope, err = t.Scope.Eval(labels, env); err != nil {
			return nil, fmt.Errorf("failed to evaluate scope: %v for relationship: %w", t.Scope, err)
		} else if output.Scope == "" {
			return nil, nil
		}
	}

	return &output, nil
}

func LookupComponents(ctx context.Context, lookup RelationshipSelectorTemplate, labels map[string]string, env map[string]any) ([]uuid.UUID, error) {
	lookupResult, err := lookup.Eval(labels, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating lookup spec: %w", err)
	} else if lookupResult == nil {
		return nil, nil
	}

	if ctx.IsTrace() {
		logger.Tracef("finding all components (%s)", lookupResult)
	}

	return query.FindComponentIDs(ctx, 0, lookupResult.ToResourceSelector())
}

func LookupConfigs(ctx context.Context, lookup RelationshipSelectorTemplate, labels map[string]string, env map[string]any) ([]uuid.UUID, error) {
	lookupResult, err := lookup.Eval(labels, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating lookup spec: %w", err)
	} else if lookupResult == nil {
		return nil, nil
	}

	if ctx.IsTrace() {
		logger.Tracef("finding all config items (%s)", lookupResult)
	}

	return query.FindConfigIDsByResourceSelector(ctx, 0, lookupResult.ToResourceSelector())
}
