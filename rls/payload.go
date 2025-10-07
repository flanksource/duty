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
}

func (s Scope) IsEmpty() bool {
	return len(s.Tags) == 0 && len(s.Agents) == 0 && len(s.Names) == 0
}

func (s Scope) Fingerprint() string {
	tagSelectors := collections.SortedMap(s.Tags)
	slices.Sort(s.Agents)
	slices.Sort(s.Names)

	data := fmt.Sprintf("agents:%s | tags:%s | names:%s", strings.Join(s.Agents, "--"), tagSelectors, strings.Join(s.Names, "--"))
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

	Disable bool `json:"disable_rls,omitempty"`
}

func (t *Payload) EvalFingerprint() {
	if t.Disable {
		t.fingerprint = "disabled"
		return
	}

	parts := []string{}
	for _, scopeArray := range [][]Scope{t.Config, t.Component, t.Playbook, t.Canary} {
		for _, scope := range scopeArray {
			if !scope.IsEmpty() {
				parts = append(parts, scope.Fingerprint())
			}
		}
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
