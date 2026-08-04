package rbac

import (
	"testing"

	"github.com/flanksource/duty/rbac/policy"
)

func TestExternalUserAliasWritesUseRPCs(t *testing.T) {
	if got := GetObjectByTable("external_user_aliases"); got != policy.ObjectDatabaseSystem {
		t.Fatalf("external_user_aliases direct access mapped to %q, want %q", got, policy.ObjectDatabaseSystem)
	}

	for _, rpc := range []string{
		"rpc/add_external_user_alias",
		"rpc/remove_external_user_alias",
		"rpc/merge_external_users",
	} {
		if got := GetObjectByTable(rpc); got != policy.ObjectCatalog {
			t.Fatalf("%s mapped to %q, want %q", rpc, got, policy.ObjectCatalog)
		}
	}
}
