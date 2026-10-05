package rbac

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
)

var testContracts = map[string][2][]string{
	policy.ActionRead:        {{policy.ResourceConfig, policy.ResourceComponent, policy.ResourceCheck, policy.ResourceCanary, policy.ResourcePlaybook, policy.ResourceConnection}, nil},
	policy.ActionPlaybookRun: {{policy.ResourcePlaybook}, {policy.ResourceConfig, policy.ResourceComponent, policy.ResourceCheck}},
}

func testContract(action string) ([]string, []string, bool) {
	c, ok := testContracts[action]
	return c[0], c[1], ok
}

func TestRoleRuleConditions(t *testing.T) {
	g := NewWithT(t)

	previous := ActionContract
	ActionContract = testContract
	t.Cleanup(func() { ActionContract = previous })

	staging, tenantA, restartPlaybooks := uuid.NewString(), uuid.NewString(), uuid.NewString()

	cfgX := models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("x")}
	cfgY := models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("y")}
	component := models.Component{ID: uuid.New(), Name: "api"}
	restartPod := models.Playbook{ID: uuid.New(), Name: "restart-pod"}
	scalePod := models.Playbook{ID: uuid.New(), Name: "scale"}

	ids := func(values ...string) []uuid.UUID {
		return lo.Map(values, func(v string, _ int) uuid.UUID { return uuid.MustParse(v) })
	}
	snapshot := membership.NewSnapshot(map[membership.Ref][]uuid.UUID{
		{Type: policy.ResourceConfig, ID: cfgX.ID}:         ids(staging, tenantA),
		{Type: policy.ResourceConfig, ID: cfgY.ID}:         ids(tenantA),
		{Type: policy.ResourceComponent, ID: component.ID}: ids(staging),
		{Type: policy.ResourcePlaybook, ID: restartPod.ID}: ids(restartPlaybooks),
		{Type: policy.ResourcePlaybook, ID: scalePod.ID}:   nil,
	})
	ctx := membership.WithSnapshot(context.New(), snapshot)

	condition := func(resource, target []string, deny bool) string {
		c, err := RuleCondition(resource, target, deny)
		g.Expect(err).ToNot(HaveOccurred())
		return c
	}

	readNarrowed := condition([]string{staging, tenantA}, nil, false)
	g.Expect(readNarrowed).To(Equal(fmt.Sprintf(
		"r.obj.Membership.Fits && !r.obj.Membership.HasTarget && '%s' in r.obj.Membership.Resource && '%s' in r.obj.Membership.Resource",
		ScopeRef(min(staging, tenantA)), ScopeRef(max(staging, tenantA)))))

	policies := fmt.Sprintf(`
p, alice, *, read, allow, %s, binding:alice
p, bob, *, playbook:run, allow, %s, binding:bob
p, carol, *, playbook:run, allow, %s, binding:carol
p, carol, *, playbook:run, deny, %s, binding:carol-deny
p, dave, *, read, allow, %s, permission:dave
`,
		readNarrowed,
		condition([]string{restartPlaybooks}, []string{staging}, false),
		condition([]string{restartPlaybooks}, nil, false),
		condition([]string{restartPlaybooks}, []string{tenantA}, true),
		lo.Must(ScopeCondition(policy.ResourceConfig, tenantA)),
	)

	enforcer, err := NewEnforcer(policies)
	g.Expect(err).ToNot(HaveOccurred())

	for _, tc := range []struct {
		name    string
		subject string
		attr    *models.ABACAttribute
		action  string
		allowed bool
	}{
		{"a resource in every Scope of the rule", "alice", &models.ABACAttribute{Config: cfgX}, policy.ActionRead, true},
		{"a resource in only some of the rule's Scopes", "alice", &models.ABACAttribute{Config: cfgY}, policy.ActionRead, false},
		{"a resource of another type, in no Scope of the rule", "alice", &models.ABACAttribute{Component: component}, policy.ActionRead, false},
		{"a request with two resources doesn't fit read", "alice", &models.ABACAttribute{Config: cfgX, Component: component}, policy.ActionRead, false},

		{"a run of a playbook in the Scope, on a target in the Scope", "bob", &models.ABACAttribute{Playbook: restartPod, Config: cfgX}, policy.ActionPlaybookRun, true},
		{"a run on a component in the target Scope", "bob", &models.ABACAttribute{Playbook: restartPod, Component: component}, policy.ActionPlaybookRun, true},
		{"a run on a target outside the Scope", "bob", &models.ABACAttribute{Playbook: restartPod, Config: cfgY}, policy.ActionPlaybookRun, false},
		{"a run without a target, by a rule that has one", "bob", &models.ABACAttribute{Playbook: restartPod}, policy.ActionPlaybookRun, false},
		{"a run of a playbook outside the Scope", "bob", &models.ABACAttribute{Playbook: scalePod, Config: cfgX}, policy.ActionPlaybookRun, false},
		{"a target's Scopes don't count for the primary resource", "bob", &models.ABACAttribute{Playbook: scalePod, Component: component}, policy.ActionPlaybookRun, false},

		{"a run without a target, by a rule without one", "carol", &models.ABACAttribute{Playbook: restartPod}, policy.ActionPlaybookRun, true},
		{"a deny rule matches the target", "carol", &models.ABACAttribute{Playbook: restartPod, Config: cfgY}, policy.ActionPlaybookRun, false},
		{"a request that doesn't fit matches every deny rule", "carol", &models.ABACAttribute{Playbook: restartPod, Config: cfgX, Component: component}, policy.ActionPlaybookRun, false},

		{"a Permission naming a Scope, by the resource's type", "dave", &models.ABACAttribute{Config: cfgY}, policy.ActionRead, true},
		{"a Permission naming a Scope, with no resource of its type", "dave", &models.ABACAttribute{Component: component}, policy.ActionRead, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			allowed, err := enforcer.Enforce(tc.subject, withMembership(ctx, tc.attr, tc.action), tc.action)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(allowed).To(Equal(tc.allowed))
		})
	}
}

func TestWithMembership(t *testing.T) {
	g := NewWithT(t)

	previous := ActionContract
	ActionContract = testContract
	t.Cleanup(func() { ActionContract = previous })

	scope := uuid.New()
	config := models.ConfigItem{ID: uuid.New()}
	playbook := models.Playbook{ID: uuid.New()}
	ctx := membership.WithSnapshot(context.New(), membership.NewSnapshot(map[membership.Ref][]uuid.UUID{
		{Type: policy.ResourceConfig, ID: config.ID}:     {scope},
		{Type: policy.ResourcePlaybook, ID: playbook.ID}: nil,
	}))

	attr := &models.ABACAttribute{Playbook: playbook, Config: config}
	run := withMembership(ctx, attr, policy.ActionPlaybookRun).Membership
	g.Expect(run.Fits).To(BeTrue())
	g.Expect(run.HasTarget).To(BeTrue())
	g.Expect(run.Resource).To(BeEmpty())
	g.Expect(run.Target).To(Equal([]any{ScopeRef(scope.String())}))
	g.Expect(run.Config).To(Equal([]any{ScopeRef(scope.String())}))
	g.Expect(attr.Membership.Fits).To(BeFalse(), "the caller's request isn't changed")

	read := withMembership(ctx, &models.ABACAttribute{Config: config}, policy.ActionRead).Membership
	g.Expect(read.Fits).To(BeTrue())
	g.Expect(read.HasTarget).To(BeFalse())
	g.Expect(read.Resource).To(Equal([]any{ScopeRef(scope.String())}))

	g.Expect(withMembership(ctx, attr, "no-contract").Membership.Fits).To(BeFalse())
	g.Expect(withMembership(ctx, attr, policy.ActionRead).Membership.Fits).To(BeFalse())

	_, err := RuleCondition(nil, nil, false)
	g.Expect(err).To(HaveOccurred())
	_, err = RuleCondition([]string{"' || true || '"}, nil, false)
	g.Expect(err).To(HaveOccurred())
}
