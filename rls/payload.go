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

	data := fmt.Sprintf("agents:%s | tags:%s | names:%s | id:%s", strings.Join(agentsCopy, "--"), tagSelectors, strings.Join(namesCopy, "--"), strings.TrimSpace(s.ID))
	return fmt.Sprintf("scope::%s", hash.Sha256Hex(data))
}

// RLS Payload that's injected postgresl parameter `request.jwt.claims`
type Payload struct {
	// cached fingerprint
	fingerprint string

	Config    []Scope `json:"config,omitempty"`
	Component []Scope `json:"component,omitempty"`
	Playbook  []Scope `json:"playbook,omitempty"`
	Canary    []Scope `json:"canary,omitempty"`
	View      []Scope `json:"view,omitempty"`

	// Scopes contains the list of scope UUIDs the user has access to.
	// This is used for generated view tables only (for now).
	Scopes []string `json:"scopes,omitempty"`

	Disable bool `json:"disable_rls,omitempty"`
}

// Get the JWT claims that'll be passed on to PostgREST
func (t Payload) JWTClaims() map[string]any {
	claims := make(map[string]any)
	if t.Disable {
		claims["disable_rls"] = true
		return claims
	}

	if len(t.Config) > 0 {
		claims["config"] = t.Config
	}

	if len(t.Component) > 0 {
		claims["component"] = t.Component
	}

	if len(t.Playbook) > 0 {
		claims["playbook"] = t.Playbook
	}

	if len(t.Canary) > 0 {
		claims["canary"] = t.Canary
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
	for _, scopeArray := range [][]Scope{t.Config, t.Component, t.Playbook, t.Canary, t.View} {
		for _, scope := range scopeArray {
			if !scope.IsEmpty() {
				parts = append(parts, scope.Fingerprint())
			}
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
