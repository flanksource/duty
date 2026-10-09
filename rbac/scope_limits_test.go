package rbac

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
)

func TestScopeLimits(t *testing.T) {
	previous := ActionContract
	ActionContract = testContract
	t.Cleanup(func() { ActionContract = previous })

	staging, production, playbooks := uuid.New(), uuid.New(), uuid.New()

	stagingCfg := models.ConfigItem{ID: uuid.New()}
	productionCfg := models.ConfigItem{ID: uuid.New()}
	otherCfg := models.ConfigItem{ID: uuid.New()}
	playbook := models.Playbook{ID: uuid.New()}
	connection := models.Connection{ID: uuid.New()}
	view := models.View{ID: uuid.New()}

	ctx := membership.WithSnapshot(context.New(), membership.NewSnapshot(map[membership.Ref][]uuid.UUID{
		{Type: policy.ResourceConfig, ID: stagingCfg.ID}:     {staging},
		{Type: policy.ResourceConfig, ID: productionCfg.ID}:  {production},
		{Type: policy.ResourceConfig, ID: otherCfg.ID}:       nil,
		{Type: policy.ResourcePlaybook, ID: playbook.ID}:     {playbooks},
		{Type: policy.ResourceConnection, ID: connection.ID}: nil,
	}))

	for _, tc := range []struct {
		name    string
		limits  [][]string
		attr    *models.ABACAttribute
		action  string
		allowed bool
	}{
		{"a config in the Scope", [][]string{{staging.String()}}, &models.ABACAttribute{Config: stagingCfg}, policy.ActionRead, true},
		{"a config outside the Scope", [][]string{{staging.String()}}, &models.ABACAttribute{Config: productionCfg}, policy.ActionRead, false},
		{"a config in either of two Scopes", [][]string{{staging.String(), production.String()}}, &models.ABACAttribute{Config: productionCfg}, policy.ActionRead, true},
		{"a config in neither of two Scopes", [][]string{{staging.String(), production.String()}}, &models.ABACAttribute{Config: otherCfg}, policy.ActionRead, false},
		{"a connection outside the Scope", [][]string{{staging.String()}}, &models.ABACAttribute{Connection: connection}, policy.ActionRead, false},
		{"a view isn't limited", [][]string{{staging.String()}}, &models.ABACAttribute{View: view}, policy.ActionRead, true},

		{"a run with the playbook and its target in the Scopes", [][]string{{staging.String(), playbooks.String()}}, &models.ABACAttribute{Playbook: playbook, Config: stagingCfg}, policy.ActionPlaybookRun, true},
		{"a run on a target outside the Scopes", [][]string{{staging.String(), playbooks.String()}}, &models.ABACAttribute{Playbook: playbook, Config: productionCfg}, policy.ActionPlaybookRun, false},
		{"a run of a playbook outside the Scopes", [][]string{{staging.String()}}, &models.ABACAttribute{Playbook: playbook, Config: stagingCfg}, policy.ActionPlaybookRun, false},

		{"two limits, in a Scope of each", [][]string{{staging.String(), production.String()}, {staging.String()}}, &models.ABACAttribute{Config: stagingCfg}, policy.ActionRead, true},
		{"two limits, in a Scope of only one", [][]string{{staging.String(), production.String()}, {staging.String()}}, &models.ABACAttribute{Config: productionCfg}, policy.ActionRead, false},
		{"a limit naming no Scope", [][]string{{}}, &models.ABACAttribute{Config: stagingCfg}, policy.ActionRead, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			limited := ctx
			for _, limit := range tc.limits {
				limited = LimitToScopes(limited, limit)
			}
			g.Expect(withinScopeLimits(limited, withMembership(limited, tc.attr, tc.action))).To(Equal(tc.allowed))
		})
	}

	t.Run("a context without limits isn't limited", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(withinScopeLimits(ctx, withMembership(ctx, &models.ABACAttribute{Config: otherCfg}, policy.ActionRead))).To(BeTrue())
	})

	t.Run("a context without the limits isn't limited", func(t *testing.T) {
		g := NewWithT(t)
		limited := LimitToScopes(ctx, []string{staging.String()})
		cleared := withoutScopeLimits(limited)
		g.Expect(ScopeLimits(cleared)).To(BeNil())
		g.Expect(withinScopeLimits(cleared, withMembership(cleared, &models.ABACAttribute{Config: productionCfg}, policy.ActionRead))).To(BeTrue())
		g.Expect(ScopeLimits(limited)).To(HaveLen(1), "the limited context isn't changed")

		g.Expect(ScopeLimits(LimitToScopes(cleared, []string{production.String()}))).To(Equal([][]string{{production.String()}}))
	})

	t.Run("a resource whose membership couldn't be read is refused", func(t *testing.T) {
		g := NewWithT(t)
		limited := LimitToScopes(context.New(), []string{staging.String()})
		g.Expect(withinScopeLimits(limited, withMembership(limited, &models.ABACAttribute{Config: stagingCfg}, policy.ActionRead))).To(BeFalse())
	})
}
