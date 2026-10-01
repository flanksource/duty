package rbac

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/policy"
)

var (
	restartPod       = models.Playbook{ID: uuid.New(), Name: "restart-pod", Namespace: "operations"}
	scaleDeployment  = models.Playbook{ID: uuid.New(), Name: "scale-deployment", Namespace: "operations"}
	stagingConfig    = models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("api"), Tags: map[string]string{"namespace": "staging", "tenant": "a"}}
	productionConfig = models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("api"), Tags: map[string]string{"namespace": "production", "tenant": "a"}}
	tenantBConfig    = models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("db"), Tags: map[string]string{"namespace": "staging", "tenant": "b"}}
	stagingComponent = models.Component{ID: uuid.New(), Name: "api", Namespace: "staging"}
	stagingCheck     = models.Check{ID: uuid.New(), Name: "http", Namespace: "staging"}
	stagingView      = models.View{ID: uuid.New(), Name: "pods", Namespace: "staging"}
)

func Test_matchRule(t *testing.T) {
	staging := RuleResources{
		policy.ResourceConfig:    {{TagSelector: "namespace=staging"}},
		policy.ResourceComponent: {{Namespace: "staging"}},
		policy.ResourceCheck:     {{Namespace: "staging"}},
	}
	tenantA := RuleResources{policy.ResourceConfig: {{TagSelector: "tenant=a"}}}

	read := RuleCondition{
		ResourceTypes: []string{policy.ResourceConfig, policy.ResourceComponent},
		Resource:      []RuleResources{staging},
	}

	readNarrowed := read
	readNarrowed.Resource = []RuleResources{staging, tenantA}

	runOnStaging := RuleCondition{
		ResourceTypes: []string{policy.ResourcePlaybook},
		Resource:      []RuleResources{{policy.ResourcePlaybook: {{Name: "restart-pod"}}}},
		TargetTypes:   []string{policy.ResourceConfig, policy.ResourceComponent, policy.ResourceCheck},
		Target:        []RuleResources{staging},
	}

	runNarrowed := runOnStaging
	runNarrowed.Target = []RuleResources{staging, tenantA}

	runWithoutTarget := runOnStaging
	runWithoutTarget.Target = nil

	unevaluable := RuleCondition{
		ResourceTypes: []string{policy.ResourceConfig},
		Resource:      []RuleResources{{policy.ResourceConfig: {{TagSelector: "namespace in (staging"}}}},
	}

	deny := func(condition RuleCondition) RuleCondition {
		condition.Deny = true
		return condition
	}

	tests := []struct {
		name      string
		attr      models.ABACAttribute
		condition RuleCondition
		want      bool
	}{
		{"a config in the scope", models.ABACAttribute{Config: stagingConfig}, read, true},
		{"a component of a scope with several types", models.ABACAttribute{Component: stagingComponent}, read, true},
		{"a config outside the scope", models.ABACAttribute{Config: productionConfig}, read, false},
		{"a type the action doesn't accept", models.ABACAttribute{View: stagingView}, read, false},
		{"no resource", models.ABACAttribute{}, read, false},
		{"two primary resources", models.ABACAttribute{Config: stagingConfig, Component: stagingComponent}, read, false},
		{"a selector that can't be evaluated", models.ABACAttribute{Config: stagingConfig}, unevaluable, false},

		{"a config in both narrowed scopes", models.ABACAttribute{Config: stagingConfig}, readNarrowed, true},
		{"a config in only the rule's scope", models.ABACAttribute{Config: tenantBConfig}, readNarrowed, false},
		{"a config in only the constraint's scope", models.ABACAttribute{Config: productionConfig}, readNarrowed, false},
		{"a component the constraint doesn't select", models.ABACAttribute{Component: stagingComponent}, readNarrowed, false},

		{"a playbook on a target in the scope", models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, runOnStaging, true},
		{"a playbook on a check in the scope", models.ABACAttribute{Playbook: restartPod, Check: stagingCheck}, runOnStaging, true},
		{"a playbook on a target outside the scope", models.ABACAttribute{Playbook: restartPod, Config: productionConfig}, runOnStaging, false},
		{"another playbook", models.ABACAttribute{Playbook: scaleDeployment, Config: stagingConfig}, runOnStaging, false},
		{"no target when the rule has one", models.ABACAttribute{Playbook: restartPod}, runOnStaging, false},
		{"no target when the rule has none", models.ABACAttribute{Playbook: restartPod}, runWithoutTarget, true},
		{"a target when the rule has none", models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, runWithoutTarget, false},
		{"two targets", models.ABACAttribute{Playbook: restartPod, Config: stagingConfig, Check: stagingCheck}, runOnStaging, false},
		{"a resource the action doesn't take", models.ABACAttribute{Playbook: restartPod, View: stagingView}, runOnStaging, false},
		{"a target in both narrowed scopes", models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, runNarrowed, true},
		{"a target outside the constraint's scope", models.ABACAttribute{Playbook: restartPod, Config: tenantBConfig}, runNarrowed, false},

		{"deny: a request outside the contract", models.ABACAttribute{Playbook: restartPod, View: stagingView}, deny(runOnStaging), true},
		{"deny: no resource", models.ABACAttribute{}, deny(read), true},
		{"deny: a selector that can't be evaluated", models.ABACAttribute{Config: stagingConfig}, deny(unevaluable), true},
		{"deny: a resource outside the scope", models.ABACAttribute{Config: productionConfig}, deny(read), false},
		{"deny: a target when the rule has none", models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, deny(runWithoutTarget), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(matchRule(&tt.attr, tt.condition)).To(Equal(tt.want))
		})
	}
}

func Test_matchRuleCasbinIntegration(t *testing.T) {
	g := NewWithT(t)

	enforcer, err := casbin.NewEnforcer("model.ini")
	g.Expect(err).ToNot(HaveOccurred())
	AddCustomFunctions(enforcer)

	condition := func(rule RuleCondition) string {
		raw, err := json.Marshal(rule)
		g.Expect(err).ToNot(HaveOccurred())
		return fmt.Sprintf("matchRule(r.obj, %q)", string(raw))
	}

	allowEverywhere := RuleCondition{
		ResourceTypes: []string{policy.ResourcePlaybook},
		Resource:      []RuleResources{{policy.ResourcePlaybook: {{Name: "*"}}}},
		TargetTypes:   []string{policy.ResourceConfig},
		Target:        []RuleResources{{policy.ResourceConfig: {{Name: "*"}}}},
	}

	denyProduction := allowEverywhere
	denyProduction.Target = []RuleResources{{policy.ResourceConfig: {{TagSelector: "namespace=production"}}}}
	denyProduction.Deny = true

	_, err = enforcer.AddPolicy("user1", "*", policy.ActionPlaybookRun, "allow", condition(allowEverywhere), "allow-rule")
	g.Expect(err).ToNot(HaveOccurred())
	_, err = enforcer.AddPolicy("user1", "*", policy.ActionPlaybookRun, "deny", condition(denyProduction), "deny-rule")
	g.Expect(err).ToNot(HaveOccurred())

	enforce := func(attr *models.ABACAttribute, action string) bool {
		allowed, err := enforcer.Enforce("user1", attr, action)
		g.Expect(err).ToNot(HaveOccurred())
		return allowed
	}

	g.Expect(enforce(&models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, policy.ActionPlaybookRun)).To(BeTrue())
	g.Expect(enforce(&models.ABACAttribute{Playbook: restartPod, Config: productionConfig}, policy.ActionPlaybookRun)).To(BeFalse())
	g.Expect(enforce(&models.ABACAttribute{Playbook: restartPod, Config: stagingConfig}, policy.ActionPlaybookApprove)).To(BeFalse())
	g.Expect(enforcer.Enforce("user1", "playbooks", policy.ActionPlaybookRun)).To(BeFalse())
}
