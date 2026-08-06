package rbac

import (
	"fmt"
	"testing"

	"github.com/flanksource/duty/rbac/policy"
	"github.com/onsi/gomega"
)

func TestExternalUserAliasWritesUseRPCs(t *testing.T) {
	g := gomega.NewWithT(t)

	got := GetObjectByTable("external_user_aliases")
	g.Expect(got).To(
		gomega.Equal(policy.ObjectDatabaseSystem),
		fmt.Sprintf("external_user_aliases direct access mapped to %q, want %q", got, policy.ObjectDatabaseSystem),
	)

	for _, rpc := range []string{
		"rpc/add_external_user_alias",
		"rpc/remove_external_user_alias",
		"rpc/merge_external_users",
	} {
		got := GetObjectByTable(rpc)
		g.Expect(got).To(
			gomega.Equal(policy.ObjectCatalog),
			fmt.Sprintf("%s mapped to %q, want %q", rpc, got, policy.ObjectCatalog),
		)
	}
}
