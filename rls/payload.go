package rls

import (
	"fmt"
	"slices"
	"strings"

	"github.com/flanksource/commons/collections"
)

// RLS Payload that's injected postgresl parameter `request.jwt.claims`
type Payload struct {
	// cached fingerprint
	fingerprint string

	Tags    []map[string]string `json:"tags,omitempty"`
	Agents  []string            `json:"agents,omitempty"`
	Disable bool                `json:"disable_rls,omitempty"`
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

	t.fingerprint = fmt.Sprintf("%s-%s", strings.Join(t.Agents, "--"), strings.Join(tagSelectors, "--"))
}

func (t *Payload) Fingerprint() string {
	if t.fingerprint == "" {
		t.EvalFingerprint()
	}

	return t.fingerprint
}
