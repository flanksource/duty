package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flanksource/commons/hash"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type WildcardResourceScope string

const (
	WildcardResourceScopeConfig    WildcardResourceScope = "config"
	WildcardResourceScopeComponent WildcardResourceScope = "component"
	WildcardResourceScopeCanary    WildcardResourceScope = "canary"
	WildcardResourceScopePlaybook  WildcardResourceScope = "playbook"
	WildcardResourceScopeView      WildcardResourceScope = "view"
)

// RLS Payload that's injected postgresl parameter `request.jwt.claims`
type Payload struct {
	// cached fingerprint
	fingerprint string

	// Scopes contains the list of scope UUIDs the user has access to.
	Scopes []uuid.UUID `json:"scopes,omitempty"`

	// WildcardScopes contains resource types that grant access to all rows of that type.
	// Wildcard scopes are not materialized directly into the table rows to avoid high writes/updates.
	// Instead, if a user has wildcard scope to a resource type, then the RLS policy matches immediately.
	WildcardScopes []WildcardResourceScope `json:"wildcard_scopes,omitempty"`

	Disable bool `json:"disable_rls,omitempty"`
}

// Get the JWT claims that'll be passed on to PostgREST
func (t Payload) JWTClaims() map[string]any {
	claims := make(map[string]any)
	if t.Disable {
		claims["disable_rls"] = true
		return claims
	}

	if len(t.Scopes) > 0 {
		claims["scopes"] = t.Scopes
	}

	if len(t.WildcardScopes) > 0 {
		claims["wildcard_scopes"] = t.WildcardScopes
	}

	return claims
}

func (t *Payload) EvalFingerprint() {
	if t.Disable {
		t.fingerprint = "disabled"
		return
	}

	parts := []string{}
	if len(t.Scopes) > 0 {
		scopesCopy := make([]string, 0, len(t.Scopes))
		for _, scope := range t.Scopes {
			scopesCopy = append(scopesCopy, scope.String())
		}
		slices.Sort(scopesCopy)
		parts = append(parts, strings.Join(scopesCopy, ","))
	}

	if len(t.WildcardScopes) > 0 {
		wildcardsCopy := make([]string, 0, len(t.WildcardScopes))
		for _, wildcard := range t.WildcardScopes {
			wildcardsCopy = append(wildcardsCopy, string(wildcard))
		}
		slices.Sort(wildcardsCopy)
		parts = append(parts, strings.Join(wildcardsCopy, ","))
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
