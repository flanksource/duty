package query_test

import (
	"testing"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gotest.tools/v3/assert"
)

func TestMatchQuery(t *testing.T) {
	config := models.ConfigItem{
		Name: lo.ToPtr("aws-demo"),
		Config: lo.ToPtr(`{
			"aws_access_key_id": "1234567890",
			"aws_secret_access_key": "1234567890"
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
		{map[string]any{"config": config.AsMap()}, "matchQuery(config, 'name=aws*')", "true"},
		{map[string]any{"config": config.AsMap()}, "matchQuery(config, 'name=azure*')", "false"},

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
	env        map[string]interface{}
	expression string
	out        string
}

func runTests(t *testing.T, tests []TestCase) {
	ctx := context.New()
	for _, tc := range tests {
		t.Run(tc.expression, func(t *testing.T) {
			out, err := ctx.RunTemplate(gomplate.Template{
				Expression: tc.expression,
			}, tc.env)

			assert.ErrorIs(t, nil, err)
			assert.Equal(t, tc.out, out)
		})
	}
}
