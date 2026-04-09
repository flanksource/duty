package changegroup

import (
	"time"

	"github.com/flanksource/duty/types"
)

// GroupingRule declares how the engine should fold matching config_changes
// into a single change_group. Rules are loaded from ScrapeConfig /
// ScrapePlugin specs (field: spec.transform.changes.grouping) and handed
// to the engine via SetRules.
type GroupingRule struct {
	Name        string   `json:"name"             yaml:"name"`
	Filter      string   `json:"filter,omitempty" yaml:"filter,omitempty"`
	Scope       Scope    `json:"scope"            yaml:"scope"`
	Window      Duration `json:"window"           yaml:"window"`
	CloseAfter  Duration `json:"closeAfter,omitempty" yaml:"closeAfter,omitempty"`
	ChangeTypes []string `json:"changeTypes,omitempty" yaml:"changeTypes,omitempty"`
	Key         string   `json:"key"              yaml:"key"`
	Details     string   `json:"details"          yaml:"details"`
	Summary     string   `json:"summary,omitempty" yaml:"summary,omitempty"`
	Priority    int      `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Compiled programs are stashed here at Validate() time so per-change
	// evaluation skips re-parsing. Not serialized.
	filterProgram  Program `json:"-" yaml:"-"`
	keyProgram     Program `json:"-" yaml:"-"`
	detailsProgram Program `json:"-" yaml:"-"`
	summaryProgram Program `json:"-" yaml:"-"`

	// literalChangeTypes is the expanded set of literal change_type strings
	// this rule matches, after pseudo-type expansion. Populated by Validate.
	literalChangeTypes map[string]struct{} `json:"-" yaml:"-"`
}

// Scope narrows which changes the rule can bind together.
type Scope struct {
	Kind  string `json:"kind"            yaml:"kind"`
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	Depth int    `json:"depth,omitempty" yaml:"depth,omitempty"`
}

const (
	ScopeSameConfig     = "same_config"
	ScopeRelated        = "related"
	ScopeAll            = "all"
	ScopeByDetailsField = "by_details_field"
)

// Duration is a time.Duration that marshals to/from a Go duration string
// (e.g. "5s", "30m") so YAML stays human-readable.
type Duration time.Duration

func (d Duration) Std() time.Duration { return time.Duration(d) }

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalYAML() (any, error) { return time.Duration(d).String(), nil }

func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// ExprKind distinguishes the expected return type of a compiled CEL expression.
type ExprKind int

const (
	ExprBool ExprKind = iota
	ExprString
	ExprGroupDetails
)

// Program is an opaque evaluator-specific compiled expression handle.
type Program any

// Env is the binding set passed to every evaluator call.
type Env struct {
	// Change is the triggering change (one-based map of the change row).
	Change map[string]any
	// Changes is the full list of members currently in the group, including
	// the triggering change (appended last). First eval: len == 1.
	Changes []map[string]any
	// Group is the current persisted ChangeGroup state or nil on first eval.
	Group map[string]any
	// Flat contains top-level shortcuts mirroring Change.* for single-change rules.
	Flat map[string]any
}

// Evaluator is implemented by an external CEL-backed evaluator (in config-db).
// duty declares only the interface to avoid pulling in CEL as a dependency.
type Evaluator interface {
	EvalBool(prog Program, env Env) (bool, error)
	EvalString(prog Program, env Env) (string, error)
	EvalGroupDetails(prog Program, env Env) (types.GroupType, error)
	Compile(expr string, kind ExprKind) (Program, error)
}

// Validate compiles every expression on the rule and expands pseudo change
// types. It must be called once when the rule is loaded. Returns an error if
// any expression fails to compile or a pseudo type is unknown.
func (r *GroupingRule) Validate(ev Evaluator) error {
	if r.Name == "" {
		return ErrEmptyRuleName
	}
	if r.Details == "" {
		return ErrMissingDetails
	}
	if r.Key == "" {
		return ErrMissingKey
	}

	literals, err := expandChangeTypes(r.ChangeTypes)
	if err != nil {
		return err
	}
	r.literalChangeTypes = literals

	if r.Filter != "" {
		p, err := ev.Compile(r.Filter, ExprBool)
		if err != nil {
			return &CompileError{Rule: r.Name, Field: "filter", Err: err}
		}
		r.filterProgram = p
	}

	p, err := ev.Compile(r.Key, ExprString)
	if err != nil {
		return &CompileError{Rule: r.Name, Field: "key", Err: err}
	}
	r.keyProgram = p

	p, err = ev.Compile(r.Details, ExprGroupDetails)
	if err != nil {
		return &CompileError{Rule: r.Name, Field: "details", Err: err}
	}
	r.detailsProgram = p

	if r.Summary != "" {
		p, err := ev.Compile(r.Summary, ExprString)
		if err != nil {
			return &CompileError{Rule: r.Name, Field: "summary", Err: err}
		}
		r.summaryProgram = p
	}

	return nil
}

// Matches reports whether the rule's change_types accept the given literal
// change_type string. An empty ChangeTypes list matches everything (after the
// filter expression is also applied).
func (r *GroupingRule) Matches(changeType string) bool {
	if len(r.literalChangeTypes) == 0 {
		return true
	}
	_, ok := r.literalChangeTypes[changeType]
	return ok
}
