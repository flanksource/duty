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

	t.Run("should compute deterministic fingerprint for Tags and Agents", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Tags: []map[string]string{
				{"z": "value1", "a": "value2"},
				{"b": "value3"},
			},
			Agents: []string{"agent2", "agent1"},
		}
		payload.EvalFingerprint()

		expectedFingerprint := "agent1--agent2-a=value2,z=value1--b=value3"
		g.Expect(payload.Fingerprint()).To(gomega.Equal(expectedFingerprint))
	})

	t.Run("should compute 'empty' fingerprint for empty Tags and Agents", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{}
		payload.EvalFingerprint()

		g.Expect(payload.Fingerprint()).To(gomega.Equal("-"))
	})

	t.Run("should sort Tags and Agents deterministically", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Tags: []map[string]string{
				{"b": "value3"},
				{"z": "value1", "a": "value2"},
			},
			Agents: []string{"agent3", "agent1", "agent2"},
		}
		payload.EvalFingerprint()

		expectedFingerprint := "agent1--agent2--agent3-a=value2,z=value1--b=value3"
		g.Expect(payload.Fingerprint()).To(gomega.Equal(expectedFingerprint))
	})

	t.Run("should cache the fingerprint after first computation", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Tags: []map[string]string{
				{"x": "value4"},
			},
			Agents: []string{"agentX"},
		}
		payload.EvalFingerprint()
		firstFingerprint := payload.Fingerprint()

		// Modify the underlying data to see if the cached fingerprint remains unchanged
		payload.Tags[0]["x"] = "modified_value"
		g.Expect(payload.Fingerprint()).To(gomega.Equal(firstFingerprint))
	})
}
