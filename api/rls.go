package api

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// RLS RLSPayload that's injected postgresl parameter `request.jwt.claims`
type RLSPayload struct {
	// cached fingerprint
	fingerprint string

	Tags    []map[string]string `json:"tags,omitempty"`
	Agents  []string            `json:"agents,omitempty"`
	Disable bool                `json:"disable_rls,omitempty"`
}

func (t *RLSPayload) EvalFingerprint() {
	if t.Disable {
		t.fingerprint = "disabled"
		return
	}

	var tagSelectors []string
	for _, t := range t.Tags {
		tagSelectors = append(tagSelectors, collections.SortedMap(t))
	}
	slices.Sort(tagSelectors)
	slices.Sort(t.Agents)

	t.fingerprint = fmt.Sprintf("%s-%s", strings.Join(t.Agents, "--"), strings.Join(tagSelectors, "--"))
}

func (t *RLSPayload) Fingerprint() string {
	if t.fingerprint == "" {
		t.EvalFingerprint()
	}

	return t.fingerprint
}

// Injects the payload as local parameter
func (t RLSPayload) SetPostgresSessionRLS(db *gorm.DB) error {
	return t.setPostgresSessionRLS(db, true)
}

// Injects the payload as sessions parameter
func (t RLSPayload) SetGlobalPostgresSessionRLS(db *gorm.DB) error {
	return t.setPostgresSessionRLS(db, false)
}

func (t RLSPayload) setPostgresSessionRLS(db *gorm.DB, local bool) error {
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
