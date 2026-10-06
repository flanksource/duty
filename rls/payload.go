package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/hash"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Scope struct {
	Tags   map[string]string `json:"tags,omitempty"`
	Agents []string          `json:"agents,omitempty"`
	Names  []string          `json:"names,omitempty"`
	ID     string            `json:"id,omitempty"`
	Deny   bool              `json:"deny,omitempty"`
}

func (s Scope) IsEmpty() bool {
	return len(s.Tags) == 0 && len(s.Agents) == 0 && len(s.Names) == 0 && strings.TrimSpace(s.ID) == ""
}

func (s Scope) Fingerprint() string {
	tagSelectors := collections.SortedMap(s.Tags)
	agentsCopy := slices.Clone(s.Agents)
	namesCopy := slices.Clone(s.Names)
	slices.Sort(agentsCopy)
	slices.Sort(namesCopy)

	data := fmt.Sprintf("agents:%s | tags:%s | names:%s | id:%s | deny:%t", strings.Join(agentsCopy, "--"), tagSelectors, strings.Join(namesCopy, "--"), strings.TrimSpace(s.ID), s.Deny)
	return fmt.Sprintf("scope::%s", hash.Sha256Hex(data))
}

// RLS Payload that's injected postgresl parameter `request.jwt.claims`
//
// Each resource type carries the subject's grants on it (see Grants). A type without grants lists no rows.
type Payload struct {
	// cached fingerprint
	fingerprint string

	Config    *Grants `json:"config,omitempty"`
	Component *Grants `json:"component,omitempty"`
	Playbook  *Grants `json:"playbook,omitempty"`
	Canary    *Grants `json:"canary,omitempty"`
	Check     *Grants `json:"check,omitempty"`

	// View filters views by their own fields. Views aren't covered by stored Scope membership.
	View []Scope `json:"view,omitempty"`

	// Scopes contains the list of scope UUIDs the user has access to.
	// This is used for generated view tables only (for now).
	Scopes []string `json:"scopes,omitempty"`

	Disable bool `json:"disable_rls,omitempty"`
}

// GrantsFor returns the grants of a resource type, e.g. "config". Nil when the type has none.
func (t *Payload) GrantsFor(resourceType string) *Grants {
	switch resourceType {
	case "config":
		return t.Config
	case "component":
		return t.Component
	case "playbook":
		return t.Playbook
	case "canary":
		return t.Canary
	case "check":
		return t.Check
	}
	return nil
}

// SetGrants sets the grants of a resource type, e.g. "config".
func (t *Payload) SetGrants(resourceType string, grants *Grants) {
	switch resourceType {
	case "config":
		t.Config = grants
	case "component":
		t.Component = grants
	case "playbook":
		t.Playbook = grants
	case "canary":
		t.Canary = grants
	case "check":
		t.Check = grants
	}
	t.fingerprint = ""
}

// GrantTypes are the resource types whose rows are filtered by grants.
var GrantTypes = []string{"config", "component", "playbook", "canary", "check"}

// Get the JWT claims that'll be passed on to PostgREST
func (t Payload) JWTClaims() map[string]any {
	claims := make(map[string]any)
	if t.Disable {
		claims["disable_rls"] = true
		return claims
	}

	for _, kind := range GrantTypes {
		if grants := t.GrantsFor(kind); grants != nil {
			claims[kind] = grants
		}
	}

	if len(t.View) > 0 {
		claims["view"] = t.View
	}

	if len(t.Scopes) > 0 {
		claims["scopes"] = t.Scopes
	}

	return claims
}

func (t *Payload) EvalFingerprint() {
	if t.Disable {
		t.fingerprint = "disabled"
		return
	}

	parts := []string{}
	for _, kind := range GrantTypes {
		if grants := t.GrantsFor(kind); grants != nil {
			parts = append(parts, kind+":"+grants.Fingerprint())
		}
	}

	for _, scope := range t.View {
		if !scope.IsEmpty() {
			parts = append(parts, "view:"+scope.Fingerprint())
		}
	}

	// Include scope UUIDs in fingerprint
	if len(t.Scopes) > 0 {
		scopesCopy := slices.Clone(t.Scopes)
		slices.Sort(scopesCopy)
		parts = append(parts, strings.Join(scopesCopy, ","))
	}

	if len(parts) == 0 {
		t.fingerprint = "empty"
		return
	}

	slices.Sort(parts)
	t.fingerprint = hash.Sha256Hex(strings.Join(parts, " | "))
}

func (t *Payload) Fingerprint() string {
	if t.fingerprint == "" {
		t.EvalFingerprint()
	}

	return t.fingerprint
}

// Injects the payload as local parameter
func (t Payload) SetPostgresSessionRLS(db *gorm.DB) error {
	return t.setPostgresSessionRLS(db, true)
}

// Injects the payload as sessions parameter
func (t Payload) SetGlobalPostgresSessionRLS(db *gorm.DB) error {
	return t.setPostgresSessionRLS(db, false)
}

func (t Payload) setPostgresSessionRLS(db *gorm.DB, local bool) error {
	rlsJSON, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshall to json: %w", err)
	}

	var scope string
	if local {
		scope = "LOCAL"
	}

	if err := db.Exec(fmt.Sprintf("SET %s ROLE postgrest_api", scope)).Error; err != nil {
		return fmt.Errorf("failed to set role: %w", err)
	}

	// NOTE: SET statements in PostgreSQL do not support parameterized queries, so we must use fmt.Sprintf
	// to inject the rlsJSON safely using pq.QuoteLiteral.
	rlsSet := fmt.Sprintf(`SET %s request.jwt.claims TO %s`, scope, pq.QuoteLiteral(string(rlsJSON)))
	if err := db.Exec(rlsSet).Error; err != nil {
		return fmt.Errorf("failed to set RLS claims: %w", err)
	}

	return nil
}
