package changegroup

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/gomega"

	"github.com/flanksource/duty/types"
)

func TestMergeAppend(t *testing.T) {
	g := gomega.NewWithT(t)

	a := uuid.New()
	b := uuid.New()
	c := uuid.New()

	existing := types.DeploymentGroup{
		Image:           "registry/app:v1",
		TargetConfigIDs: []uuid.UUID{a, b},
	}
	incoming := types.DeploymentGroup{
		Image:           "registry/app:v1",
		TargetConfigIDs: []uuid.UUID{b, c}, // b is a duplicate
	}

	out, err := Merge(existing, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	dep := out.(types.DeploymentGroup)
	g.Expect(dep.TargetConfigIDs).To(gomega.Equal([]uuid.UUID{a, b, c}))
	g.Expect(dep.Image).To(gomega.Equal("registry/app:v1"))
}

func TestMergeFirstSet(t *testing.T) {
	g := gomega.NewWithT(t)

	grant := uuid.New()
	revoke := uuid.New()
	otherGrant := uuid.New()

	existing := types.TemporaryPermissionGroup{
		UserID:        "u1",
		GrantChangeID: &grant,
	}
	// "incoming" arrives with a new revoke and a (wrong) new grant id — firstSet
	// must keep the old grant id.
	incoming := types.TemporaryPermissionGroup{
		UserID:         "u1",
		GrantChangeID:  &otherGrant,
		RevokeChangeID: &revoke,
	}

	out, err := Merge(existing, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	tp := out.(types.TemporaryPermissionGroup)
	g.Expect(tp.GrantChangeID).To(gomega.Equal(&grant))
	g.Expect(tp.RevokeChangeID).To(gomega.Equal(&revoke))
}

func TestMergeMinMax(t *testing.T) {
	g := gomega.NewWithT(t)

	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	existing := types.IncidentResponseGroup{
		IncidentID:     "INC-1",
		OpenedAt:       t2,
		ClosedAt:       t2,
		PlaybookRunIDs: []uuid.UUID{uuid.New()},
	}
	incoming := types.IncidentResponseGroup{
		IncidentID:     "INC-1",
		OpenedAt:       t1, // earlier → wins min
		ClosedAt:       t3, // later → wins max
		PlaybookRunIDs: []uuid.UUID{uuid.New()},
	}

	out, err := Merge(existing, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	ir := out.(types.IncidentResponseGroup)
	g.Expect(ir.OpenedAt).To(gomega.Equal(t1))
	g.Expect(ir.ClosedAt).To(gomega.Equal(t3))
	g.Expect(ir.PlaybookRunIDs).To(gomega.HaveLen(2))
}

func TestMergeIgnoresZeroIncoming(t *testing.T) {
	g := gomega.NewWithT(t)

	existing := types.DeploymentGroup{
		Image:   "registry/app:v1",
		Version: "v1",
	}
	// Incoming omits Version — existing Version must survive.
	incoming := types.DeploymentGroup{
		Image: "registry/app:v2",
	}

	out, err := Merge(existing, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	dep := out.(types.DeploymentGroup)
	g.Expect(dep.Image).To(gomega.Equal("registry/app:v2"), "scalar last-write-wins")
	g.Expect(dep.Version).To(gomega.Equal("v1"), "zero incoming must not clobber")
}

func TestMergeKindMismatch(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := Merge(
		types.DeploymentGroup{Image: "a"},
		types.PromotionGroup{Version: "v1"},
	)
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestMergeCustomGroup(t *testing.T) {
	g := gomega.NewWithT(t)

	existing := types.CustomGroup{Fields: map[string]any{"a": 1, "b": 2}}
	incoming := types.CustomGroup{Fields: map[string]any{"b": 20, "c": 3}}

	out, err := Merge(existing, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	cg := out.(types.CustomGroup)
	g.Expect(cg.Fields).To(gomega.Equal(map[string]any{"a": 1, "b": 20, "c": 3}))
}

func TestMergeNilFirstMember(t *testing.T) {
	g := gomega.NewWithT(t)

	incoming := types.DeploymentGroup{Image: "a"}
	out, err := Merge(nil, incoming)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(out).To(gomega.Equal(incoming))
}
