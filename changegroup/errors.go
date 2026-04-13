package changegroup

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyRuleName    = errors.New("changegroup: rule name is required")
	ErrMissingDetails   = errors.New("changegroup: rule details expression is required")
	ErrMissingKey       = errors.New("changegroup: rule key expression is required")
	ErrUnknownPseudo    = errors.New("changegroup: unknown pseudo change type")
	ErrMissingEvaluator = errors.New("changegroup: evaluator must be set before rules are loaded")
)

// EvalError wraps an evaluator runtime failure with the originating rule and field.
type EvalError struct {
	Rule  string
	Field string
	Err   error
}

func (e *EvalError) Error() string {
	return fmt.Sprintf("changegroup: eval rule %q field %q: %v", e.Rule, e.Field, e.Err)
}

func (e *EvalError) Unwrap() error { return e.Err }

// CompileError wraps an evaluator compile failure with the originating rule
// and field so operators can pinpoint their YAML.
type CompileError struct {
	Rule  string
	Field string
	Err   error
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("changegroup: compile rule %q field %q: %v", e.Rule, e.Field, e.Err)
}

func (e *CompileError) Unwrap() error { return e.Err }
