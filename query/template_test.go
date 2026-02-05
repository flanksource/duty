package query_test

import (
	"fmt"
	"testing"

	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

func TestMatchQuery(t *testing.T) {
	configItemID := uuid.New()
	agentID := uuid.New()
	parentID := uuid.New()

	configItem := models.ConfigItem{
		ID:          configItemID,
		AgentID:     agentID,
		ParentID:    &parentID,
		ConfigClass: "Deployment",
		Type:        lo.ToPtr("Kubernetes::Deployment"),
		Status:      lo.ToPtr("Running"),
		Ready:       true,
		Health:      lo.ToPtr(models.HealthHealthy),
		Name:        lo.ToPtr("my-app"),
		Description: lo.ToPtr("Main application deployment"),
		Tags: types.JSONStringMap{
			"namespace":   "production",
			"team":        "backend",
			"version":     "v1.2.3",
			"environment": "prod",
			"cost-center": "engineering",
		},
		Labels: &types.JSONStringMap{
			"app.kubernetes.io/name":            "my-app",
			"app.kubernetes.io/version":         "1.2.3",
			"app.kubernetes.io/component":       "backend",
			"deployment.kubernetes.io/revision": "42",
		},
		Properties: &types.Properties{
			{Name: "cpu", Text: "2000m"},
			{Name: "memory", Text: "4Gi"},
			{Name: "replicas", Value: lo.ToPtr(int64(3))},
			{Name: "maxReplicas", Value: lo.ToPtr(int64(10))},
		},
		Config: lo.ToPtr(`{
			"apiVersion": "apps/v1",
			"kind": "Deployment",
			"metadata": {
				"name": "my-app",
				"namespace": "production"
			},
			"spec": {
				"replicas": 3,
				"strategy": {
					"type": "RollingUpdate"
				}
			}
		}`),
	}

	playbook := models.Playbook{
		Name:     "airsonic",
		Category: "kubernetes",
	}

	check := models.Check{
		Name:     "webhook",
		CanaryID: uuid.New(),
	}

	component := models.Component{
		Name:       "azure-demo",
		AgentID:    uuid.New(),
		TopologyID: lo.ToPtr(uuid.New()),
		Namespace:  "mission-control",
	}

	runTests(t, []TestCase{
		// Basic field matching tests
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my-app')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=*y-app*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=other-app')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=*app')", "true"},

		// ID matching
		{map[string]any{"config": configItem.AsMap()}, fmt.Sprintf("matchQuery(config, 'id=%s')", configItemID.String()), "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'id=00000000-0000-0000-0000-000000000000')", "false"},

		// Type matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'type=Kubernetes::Deployment')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'type=Kubernetes*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'type=*Deployment')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'type=Docker::Container')", "false"},

		// Status matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'status=Running')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'status=Run*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'status=Failed')", "false"},

		// Health matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'health=healthy')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'health=heal*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'health=unhealthy')", "false"},

		// Tags matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.team=backend')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.team=back*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.team=frontend')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.version=v1.2.3')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.version=v1.*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.environment=prod')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.cost-center=engineering')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.nonexistent=value')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.team')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!tags.team')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.nonexistent')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!tags.nonexistent')", "true"},

		// Labels matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/name=my-app')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/name=my*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/version=1.2.3')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/component=backend')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.deployment.kubernetes.io/revision=42')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.nonexistent=value')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/name')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!labels.app.kubernetes.io/name')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.account')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!labels.account')", "true"},

		// Properties matching
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.cpu=2000m')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.cpu=2000*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.memory=4Gi')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.replicas=3')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.maxReplicas=10')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.nonexistent=value')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.cpu')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!properties.cpu')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.nonexistent')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, '!properties.nonexistent')", "true"},

		// Multiple field combinations
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my-app type=Kubernetes::Deployment')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my-app status=Running,health=healthy')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.namespace=production tags.team=backend')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/name=my-app tags.version=v1.2.3')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'properties.replicas=3 properties.cpu=2000m')", "true"},

		// Mixed positive and negative combinations - logic incorrect
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my-app type=Docker::Container')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'status=Running health=unhealthy')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'namespace=staging tags.team=backend')", "false"},

		// Wildcard combinations
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=my*,type=Kubernetes*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'tags.team=back*,tags.version=v1*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'labels.app.kubernetes.io/name=*app,properties.cpu=*m')", "true"},

		// Edge cases with empty/missing values
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'name=')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'nonexistent=value')", "false"},

		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'config_class=Deployment')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'config_class=Deploy*')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'config_class=Service')", "false"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'ready=true')", "true"},
		{map[string]any{"config": configItem.AsMap()}, "matchQuery(config, 'ready=false')", "false"},

		{map[string]any{"playbook": playbook.AsMap()}, "matchQuery(playbook, 'name=air*')", "true"},
		{map[string]any{"playbook": playbook.AsMap()}, "matchQuery(playbook, 'name=azure*')", "false"},

		{map[string]any{"check": check.AsMap()}, "matchQuery(check, 'name=web*')", "true"},
		{map[string]any{"check": check.AsMap()}, "matchQuery(check, 'name=azure*')", "false"},

		{map[string]any{"component": component.AsMap()}, "matchQuery(component, 'name=azure*')", "true"},
		{map[string]any{"component": component.AsMap()}, "matchQuery(component, 'namespace=*control')", "true"},
		{map[string]any{"component": component.AsMap()}, "matchQuery(component, 'namespace=default')", "false"},

		{map[string]any{"generic": map[string]any{"name": "navidrome", "namespace": "music"}}, "matchQuery(generic, 'name=navidrome,namespace=music')", "true"},
		{map[string]any{"generic": map[string]any{"name": "navidrome", "namespace": "music"}}, "matchQuery(generic, 'name=airsonic,namespace=music')", "false"},
	})
}

type TestCase struct {
	env        map[string]any
	expression string
	out        string
}

func runTests(t *testing.T, tests []TestCase) {
	ctx := context.New()
	for _, tc := range tests {
		t.Run(tc.expression, func(t *testing.T) {
			g := gomega.NewWithT(t)
			out, err := ctx.RunTemplate(gomplate.Template{
				Expression: tc.expression,
			}, tc.env)

			g.Expect(err).To(gomega.BeNil())
			g.Expect(out).To(gomega.Equal(tc.out))
		})
	}
}
