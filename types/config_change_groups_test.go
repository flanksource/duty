package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
)

func TestGroupDetailsRoundTrip(t *testing.T) {
	cfg := uuid.New()
	grant := uuid.New()
	revoke := uuid.New()
	dur := int64(3600)
	now := time.Now().UTC().Truncate(time.Second)

	cases := []struct {
		name string
		in   GroupType
	}{
		{
			name: "startup",
			in:   StartupGroup{ConfigID: cfg, Reason: "CrashLoopBackOff", RestartCount: 3},
		},
		{
			name: "deployment",
			in: DeploymentGroup{
				Image:           "registry/app:v1.2.3",
				Version:         "v1.2.3",
				Commit:          "abc123",
				Strategy:        "RollingUpdate",
				TargetConfigIDs: []uuid.UUID{uuid.New(), uuid.New()},
			},
		},
		{
			name: "promotion",
			in: PromotionGroup{
				FromEnvironment:     "dev",
				ToEnvironment:       "prod",
				Version:             "v1.2.3",
				Artifact:            "app",
				PromotionChangeID:   &grant,
				TargetDeploymentIDs: []uuid.UUID{uuid.New()},
			},
		},
		{
			name: "temporary_permission",
			in: TemporaryPermissionGroup{
				UserID:          "user-1",
				RoleID:          "admin",
				Scope:           "cluster-1",
				GrantChangeID:   &grant,
				RevokeChangeID:  &revoke,
				DurationSeconds: &dur,
			},
		},
		{
			name: "incident_response",
			in: IncidentResponseGroup{
				IncidentID:     "INC-42",
				OpenedAt:       now,
				ClosedAt:       now.Add(time.Hour),
				PlaybookRunIDs: []uuid.UUID{uuid.New()},
			},
		},
		{
			name: "custom",
			in:   CustomGroup{Fields: map[string]any{"foo": "bar", "n": float64(42)}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			raw, err := json.Marshal(tc.in)
			g.Expect(err).ToNot(gomega.HaveOccurred())

			// envelope must carry the kind.
			var envelope struct {
				Kind string `json:"kind"`
			}
			g.Expect(json.Unmarshal(raw, &envelope)).To(gomega.Succeed())
			g.Expect(envelope.Kind).To(gomega.Equal(tc.in.Kind()))

			got, err := UnmarshalGroupDetails(raw)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(got.Kind()).To(gomega.Equal(tc.in.Kind()))

			// Round-tripped value re-marshals to the same JSON.
			reraw, err := json.Marshal(got)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(reraw).To(gomega.MatchJSON(raw))
		})
	}
}

func TestUnmarshalGroupDetailsEmpty(t *testing.T) {
	g := gomega.NewWithT(t)

	got, err := UnmarshalGroupDetails(nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got).To(gomega.BeNil())

	got, err = UnmarshalGroupDetails(json.RawMessage("null"))
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got).To(gomega.BeNil())
}

func TestUnmarshalGroupDetailsUnknownKind(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := UnmarshalGroupDetails(json.RawMessage(`{"kind":"Nope/v1"}`))
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("unknown group kind")))
}
