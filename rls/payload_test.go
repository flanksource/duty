package rls

import (
	"testing"

	"github.com/google/uuid"
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
			Scopes: []uuid.UUID{
				uuid.MustParse("b6e3e8b2-8cda-4b70-bde7-3fb48c36d3f2"),
				uuid.MustParse("0a1ce1b2-5d90-4e74-8d30-2f4f0d30f8e4"),
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
			Scopes: []uuid.UUID{
				uuid.MustParse("0a1ce1b2-5d90-4e74-8d30-2f4f0d30f8e4"),
				uuid.MustParse("b6e3e8b2-8cda-4b70-bde7-3fb48c36d3f2"),
			},
		}
		payload1.EvalFingerprint()

		payload2 := &Payload{
			Scopes: []uuid.UUID{
				uuid.MustParse("b6e3e8b2-8cda-4b70-bde7-3fb48c36d3f2"),
				uuid.MustParse("0a1ce1b2-5d90-4e74-8d30-2f4f0d30f8e4"),
			},
		}
		payload2.EvalFingerprint()

		// Same scopes in different order should produce the same fingerprint
		g.Expect(payload1.Fingerprint()).To(gomega.Equal(payload2.Fingerprint()))
	})

	t.Run("should cache the fingerprint after first computation", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{
			Scopes: []uuid.UUID{uuid.MustParse("b6e3e8b2-8cda-4b70-bde7-3fb48c36d3f2")},
		}
		payload.EvalFingerprint()
		firstFingerprint := payload.Fingerprint()

		// Modify the underlying data to see if the cached fingerprint remains unchanged
		payload.Scopes[0] = uuid.MustParse("f4a1fcb2-4cf7-48f2-9e68-6457e8c4e9e6")
		g.Expect(payload.Fingerprint()).To(gomega.Equal(firstFingerprint))
	})
}
