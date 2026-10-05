package rls

import (
	"encoding/json"
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

	t.Run("should compute 'empty' fingerprint for empty payload", func(t *testing.T) {
		g := gomega.NewWithT(t)

		payload := &Payload{}
		payload.EvalFingerprint()

		g.Expect(payload.Fingerprint()).To(gomega.Equal("empty"))
	})

	t.Run("should not depend on the order grants were added in", func(t *testing.T) {
		g := gomega.NewWithT(t)

		a := &Grants{}
		a.Add("s1", "s2")
		a.Add("s3")
		b := &Grants{}
		b.Add("s3")
		b.Add("s2", "s1")

		g.Expect((&Payload{Config: a}).Fingerprint()).To(gomega.Equal((&Payload{Config: b}).Fingerprint()))
	})

	t.Run("should tell types, all and no rows apart", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add("s1")

		fingerprints := []string{
			(&Payload{Config: grants}).Fingerprint(),
			(&Payload{Component: grants}).Fingerprint(),
			(&Payload{Config: AllRows()}).Fingerprint(),
			(&Payload{Config: NoRows()}).Fingerprint(),
			(&Payload{}).Fingerprint(),
		}
		for i := range fingerprints {
			for j := range fingerprints {
				if i != j {
					g.Expect(fingerprints[i]).NotTo(gomega.Equal(fingerprints[j]))
				}
			}
		}
	})
}

func TestGrants(t *testing.T) {
	t.Run("marshals all rows as \"all\" and grants as sets of Scopes", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add("B", "a", "a")
		grants.Add("c")
		grants.Add("a", "b")
		grants.Add()

		raw, err := json.Marshal(Payload{Config: AllRows(), Component: grants, Check: NoRows()})
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(raw)).To(gomega.MatchJSON(`{"config":"all","component":[["a","b"],["c"]],"check":[]}`))

		var decoded Payload
		g.Expect(json.Unmarshal(raw, &decoded)).To(gomega.Succeed())
		g.Expect(decoded.Config.All).To(gomega.BeTrue())
		g.Expect(decoded.Component.Sets).To(gomega.Equal([][]string{{"a", "b"}, {"c"}}))
		g.Expect(decoded.Check.IsEmpty()).To(gomega.BeTrue())
		g.Expect(decoded.Playbook).To(gomega.BeNil())
	})

	t.Run("narrows every grant, and all rows to the narrowing Scopes", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add("a")
		grants.Add("b", "c")
		grants.Narrow("x")
		g.Expect(grants.Sets).To(gomega.Equal([][]string{{"a", "x"}, {"b", "c", "x"}}))

		all := AllRows()
		all.Narrow("x", "y")
		g.Expect(all.All).To(gomega.BeFalse())
		g.Expect(all.Sets).To(gomega.Equal([][]string{{"x", "y"}}))

		none := NoRows()
		none.Narrow("x")
		g.Expect(none.IsEmpty()).To(gomega.BeTrue())
	})

	t.Run("rejects anything but all or a list of grants", func(t *testing.T) {
		g := gomega.NewWithT(t)

		var grants Grants
		g.Expect(json.Unmarshal([]byte(`"some"`), &grants)).ToNot(gomega.Succeed())
		g.Expect(json.Unmarshal([]byte(`{"tags":{}}`), &grants)).ToNot(gomega.Succeed())
	})
}
