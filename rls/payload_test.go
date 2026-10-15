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
		a.Add(Grant{Scope: s1, Constraint: s2})
		a.Add(Grant{Scope: s3})
		b := &Grants{}
		b.Add(Grant{Scope: s3})
		b.Add(Grant{Scope: s1, Constraint: s2})

		g.Expect((&Payload{Config: a}).Fingerprint()).To(gomega.Equal((&Payload{Config: b}).Fingerprint()))
	})

	t.Run("should tell types, all and no rows apart", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add(Grant{Scope: s1})

		fingerprints := []string{
			(&Payload{Config: grants}).Fingerprint(),
			(&Payload{Component: grants}).Fingerprint(),
			(&Payload{Scraper: grants}).Fingerprint(),
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

	t.Run("should tell a constraint from an impersonated Scope", func(t *testing.T) {
		g := gomega.NewWithT(t)

		constrained := &Grants{}
		constrained.Add(Grant{Scope: s1, Constraint: s2})
		impersonated := &Grants{}
		impersonated.Add(Grant{Scope: s1, Impersonated: []string{s2}})

		g.Expect((&Payload{Config: constrained}).Fingerprint()).NotTo(gomega.Equal((&Payload{Config: impersonated}).Fingerprint()))
	})
}

const (
	s1 = "00000000-0000-0000-0000-0000000000a1"
	s2 = "00000000-0000-0000-0000-0000000000a2"
	s3 = "00000000-0000-0000-0000-0000000000a3"
	x  = "00000000-0000-0000-0000-0000000000b1"
	y  = "00000000-0000-0000-0000-0000000000b2"
)

func TestGrants(t *testing.T) {
	t.Run("marshals all rows as \"all\" and grants as a list of grants", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add(Grant{Scope: s3})
		grants.Add(Grant{Scope: s1, Constraint: s2})
		grants.Add(Grant{Scope: s1, Constraint: s2})

		raw, err := json.Marshal(Payload{Config: AllRows(), Component: grants, Check: NoRows(), Scraper: grants})
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(raw)).To(gomega.MatchJSON(`{
			"config": "all",
			"component": [{"scope": "` + s1 + `", "constraint": "` + s2 + `"}, {"scope": "` + s3 + `"}],
			"check": [],
			"scraper": [{"scope": "` + s1 + `", "constraint": "` + s2 + `"}, {"scope": "` + s3 + `"}]
		}`))

		var decoded Payload
		g.Expect(json.Unmarshal(raw, &decoded)).To(gomega.Succeed())
		g.Expect(decoded.Config.All).To(gomega.BeTrue())
		g.Expect(decoded.Component.Any).To(gomega.Equal([]Grant{{Scope: s1, Constraint: s2}, {Scope: s3}}))
		g.Expect(decoded.Check.IsEmpty()).To(gomega.BeTrue())
		g.Expect(decoded.Scraper.Any).To(gomega.Equal(decoded.Component.Any))
		g.Expect(decoded.Playbook).To(gomega.BeNil())

		claims := (Payload{Scraper: AllRows()}).JWTClaims()
		g.Expect(claims).To(gomega.HaveKeyWithValue("scraper", AllRows()))
	})

	t.Run("normalizes a grant, and ignores one that isn't made of Scope ids", func(t *testing.T) {
		g := gomega.NewWithT(t)

		grants := &Grants{}
		grants.Add(Grant{Scope: " 00000000-0000-0000-0000-0000000000A1 ", Constraint: s1})
		grants.Add(Grant{Scope: s2, Constraint: s3, Impersonated: []string{s3, x, x}})
		grants.Add(Grant{Scope: "{00000000-0000-0000-0000-0000000000a3}"})
		grants.Add(Grant{Scope: "000000000000000000000000000000a3"})
		grants.Add(Grant{Constraint: s1})
		grants.Add(Grant{Scope: "staging"})
		grants.Add(Grant{Scope: s1, Constraint: "staging"})
		grants.Add(Grant{Scope: s1, Impersonated: []string{"staging"}})
		grants.Add(Grant{})

		g.Expect(grants.Any).To(gomega.Equal([]Grant{
			{Scope: s1},
			{Scope: s2, Constraint: s3, Impersonated: []string{x}},
			{Scope: s3},
		}))
		g.Expect(grants.ScopeIDs()).To(gomega.Equal([]string{s1, s2, s3, x}))
	})

	t.Run("rejects anything but all or a list of grants", func(t *testing.T) {
		g := gomega.NewWithT(t)

		var grants Grants
		g.Expect(json.Unmarshal([]byte(`"some"`), &grants)).ToNot(gomega.Succeed())
		g.Expect(json.Unmarshal([]byte(`{"tags":{}}`), &grants)).ToNot(gomega.Succeed())
		g.Expect(json.Unmarshal([]byte(`[["`+s1+`"]]`), &grants)).ToNot(gomega.Succeed())
	})
}
