package changegroup

import (
	"errors"
	"testing"

	"github.com/onsi/gomega"

	"github.com/flanksource/duty/types"
)

func TestExpandPseudo(t *testing.T) {
	cases := []struct {
		pseudo string
		want   []string
	}{
		{
			pseudo: PseudoCreated,
			want: []string{
				"Created",
				types.ChangeTypeCreate,
				types.ChangeTypeRegisterNode,
				types.ChangeTypeRunInstances,
				types.ChangeTypeUserCreated,
			},
		},
		{
			pseudo: PseudoUnhealthy,
			want: []string{
				types.ChangeTypeBackupFailed,
				types.ChangeTypeCertificateExpired,
				types.ChangeTypePipelineRunFailed,
				types.ChangeTypePlaybookFailed,
				"Unhealthy",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.pseudo, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := ExpandPseudo(tc.pseudo)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(got).To(gomega.ConsistOf(tc.want))
		})
	}
}

func TestExpandPseudoUnknown(t *testing.T) {
	g := gomega.NewWithT(t)
	_, err := ExpandPseudo("@nope")
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(errors.Is(err, ErrUnknownPseudo)).To(gomega.BeTrue())
}

func TestExpandChangeTypesMixed(t *testing.T) {
	g := gomega.NewWithT(t)

	got, err := expandChangeTypes([]string{
		"PermissionAdded",
		"@unhealthy",
		"PermissionAdded", // dedupe
		"",                // skip
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(got).To(gomega.HaveKey("PermissionAdded"))
	g.Expect(got).To(gomega.HaveKey(types.ChangeTypeBackupFailed))
	g.Expect(got).To(gomega.HaveKey(types.ChangeTypePlaybookFailed))
	g.Expect(got).ToNot(gomega.HaveKey(""))
}

func TestExpandChangeTypesUnknownPseudoPropagates(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := expandChangeTypes([]string{"Deployment", "@foo"})
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(errors.Is(err, ErrUnknownPseudo)).To(gomega.BeTrue())
}

func TestExpandChangeTypesEmptyAcceptsAll(t *testing.T) {
	g := gomega.NewWithT(t)

	got, err := expandChangeTypes(nil)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(got).To(gomega.BeEmpty())

	// Matches helper treats empty set as "accept everything".
	r := &GroupingRule{literalChangeTypes: got}
	g.Expect(r.Matches("Literally anything")).To(gomega.BeTrue())
}
