package duty

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

// Lookup specifies the type of lookup to perform.
type Lookup struct {
	// Expr is a cel-expression.
	Expr string `json:"expr,omitempty" yaml:"expr,omitempty"`
	// Value is the static value to use.
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	// Label specifies the key to lookup on the label.
	Label string `json:"label,omitempty" yaml:"label,omitempty"`
}

func (t *Lookup) Empty() bool {
	return t.Expr == "" && t.Value == "" && t.Label == ""
}

func (t *Lookup) Eval(labels map[string]string, envVar map[string]any) (string, error) {
	if t.Empty() {
		return "", nil
	}

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

// LookupSpec defines a tuple of fields to lookup.
type LookupSpec struct {
	Name      Lookup `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace Lookup `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Type      Lookup `json:"type,omitempty" yaml:"type,omitempty"`
}

// LookupEvalResult is the result of evaluation of a LookupSpec.
type LookupEvalResult struct {
	Name      string
	Namespace string
	Type      string
}

// Eval evaluates all the fields in the lookup spec based on labels and environment variables.
// Returns nil if any non-empty lookup evaluates to an empty value.
func (t *LookupSpec) Eval(labels map[string]string, envVar map[string]any) (*LookupEvalResult, error) {
	var result LookupEvalResult

	if !t.Name.Empty() {
		name, err := t.Name.Eval(labels, envVar)
		if err != nil {
			return nil, err
		}
		if name == "" {
			return nil, nil
		}
		result.Name = name
	}

	if !t.Namespace.Empty() {
		namespace, err := t.Namespace.Eval(labels, envVar)
		if err != nil {
			return nil, err
		}
		if namespace == "" {
			return nil, nil
		}
		result.Namespace = namespace
	}

	if !t.Type.Empty() {
		typ, err := t.Type.Eval(labels, envVar)
		if err != nil {
			return nil, err
		}
		if typ == "" {
			return nil, nil
		}
		result.Type = typ
	}

	return &result, nil
}

func LookupComponents(ctx context.Context, lookup LookupSpec, labels map[string]string, env map[string]any) ([]uuid.UUID, error) {
	lookupResult, err := lookup.Eval(labels, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating lookup spec: %w", err)
	} else if lookupResult == nil {
		return nil, nil
	}

	if ctx.IsTrace() {
		logger.Tracef("Finding all components (namespace=%s) (name=%s) (type=%s)", lookupResult.Namespace, lookupResult.Name, lookupResult.Type)
	}
	return query.FindComponentIDs(ctx, types.ResourceSelector{
		Namespace: lookupResult.Namespace,
		Name:      lookupResult.Name,
		Types:     []string{lookupResult.Type},
	})
}

func LookupConfigs(ctx context.Context, lookup LookupSpec, labels map[string]string, env map[string]any) ([]uuid.UUID, error) {
	lookupResult, err := lookup.Eval(labels, env)
	if err != nil {
		return nil, fmt.Errorf("error evaluating lookup spec: %w", err)
	} else if lookupResult == nil {
		return nil, nil
	}

	if ctx.IsTrace() {
		logger.Tracef("Finding all config items (namespace=%s) (name=%s) (type=%s)", lookupResult.Namespace, lookupResult.Name, lookupResult.Type)
	}

	return query.FindConfigIDsByResourceSelector(ctx, types.ResourceSelector{
		Namespace: lookupResult.Namespace,
		Name:      lookupResult.Name,
		Types:     []string{lookupResult.Type},
	})
}
