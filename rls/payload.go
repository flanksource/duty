package rls

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/flanksource/duty/types"
)

// ObjectSelectors contains resource selectors for different object types
type ObjectSelectors struct {
	Playbooks   []types.ResourceSelector `json:"playbooks,omitempty"`
	Connections []types.ResourceSelector `json:"connections,omitempty"`
	Configs     []types.ResourceSelector `json:"configs,omitempty"`
	Components  []types.ResourceSelector `json:"components,omitempty"`
}

// RLS Payload that's injected postgresl parameter `request.jwt.claims`
type Payload struct {
	// cached fingerprint
	fingerprint string

	Tags            []map[string]string `json:"tags,omitempty"`
	Agents          []string            `json:"agents,omitempty"`
	Objects         []string            `json:"objects,omitempty"`
	ObjectSelectors *ObjectSelectors    `json:"object_selectors,omitempty"`
	Disable         bool                `json:"disable_rls,omitempty"`
}

func (t *Payload) EvalFingerprint() {
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
	slices.Sort(t.Objects)

	var objectSelectorsStr string
	if t.ObjectSelectors != nil {
		if data, err := json.Marshal(t.ObjectSelectors); err == nil {
			objectSelectorsStr = string(data)
		}
	}

	t.fingerprint = fmt.Sprintf("%s-%s-%s-%s",
		strings.Join(t.Agents, "--"),
		strings.Join(tagSelectors, "--"),
		strings.Join(t.Objects, "--"),
		objectSelectorsStr,
	)
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
