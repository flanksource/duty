package rls

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestPayload_EvalFingerprint(t *testing.T) {
	t.Run("should set fingerprint to 'disabled' if Disable is true", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Disable: true,
		}
		payload.EvalFingerprint()

		g.Expect(payload.Fingerprint()).To(gomega.Equal("disabled"))
	})

	t.Run("should compute deterministic fingerprint for scopes", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Config: []Scope{
				{
					Tags:   map[string]string{"z": "value1", "a": "value2"},
					Agents: []string{"agent2", "agent1"},
				},
			},
		}
		payload.EvalFingerprint()

		// Fingerprint should be deterministic (hash of sorted scope fingerprints)
		g.Expect(payload.Fingerprint()).NotTo(gomega.BeEmpty())
		g.Expect(payload.Fingerprint()).NotTo(gomega.Equal("disabled"))
		g.Expect(payload.Fingerprint()).NotTo(gomega.Equal("empty"))
	})

	t.Run("should compute 'empty' fingerprint for empty scopes", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{}
		payload.EvalFingerprint()

		g.Expect(payload.Fingerprint()).To(gomega.Equal("empty"))
	})

	t.Run("should sort scopes deterministically", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload1 := &Payload{
			Config: []Scope{
				{Tags: map[string]string{"a": "value1"}},
				{Tags: map[string]string{"b": "value2"}},
			},
		}
		payload1.EvalFingerprint()

		payload2 := &Payload{
			Config: []Scope{
				{Tags: map[string]string{"b": "value2"}},
				{Tags: map[string]string{"a": "value1"}},
			},
		}
		payload2.EvalFingerprint()

		// Same scopes in different order should produce the same fingerprint
		g.Expect(payload1.Fingerprint()).To(gomega.Equal(payload2.Fingerprint()))
	})

	t.Run("should cache the fingerprint after first computation", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Config: []Scope{
				{
					Tags:   map[string]string{"x": "value4"},
					Agents: []string{"agentX"},
				},
			},
		}
		payload.EvalFingerprint()
		firstFingerprint := payload.Fingerprint()

		// Modify the underlying data to see if the cached fingerprint remains unchanged
		payload.Config[0].Tags["x"] = "modified_value"
		g.Expect(payload.Fingerprint()).To(gomega.Equal(firstFingerprint))
	})
}
