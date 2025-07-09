package dummy

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var View = models.View{
	ID:        uuid.New(),
	Name:      "Mission Control",
	Namespace: "default",
	Spec: types.JSON([]byte(`{
		"cacheTTL": "1h",
		"panels": [
			{
				"name": "Average Duration",
				"description": "Create Release average duration",
				"type": "number",
				"source": "changes",
				"number": {
					"unit": "seconds"
				},
				"query": {
					"search": "change_type=GitHubActionRun*",
					"name": "Create Release",
					"types": [
						"GitHubAction::Workflow"
					],
					"groupBy": [
						"details.repository.full_name"
					],
					"aggregates": [
						{
							"function": "AVG",
							"alias": "value",
							"field": "details.duration"
						}
					]
				}
			}
		]
	}`)),
	Source:    "KubernetesCRD",
	CreatedBy: lo.ToPtr(JohnDoe.ID),
	CreatedAt: DummyCreatedAt,
}

var PipelineView = models.ViewPanel{
	ViewID:   View.ID,
	AgentID:  uuid.Nil,
	IsPushed: false,
	Results: types.JSON([]byte(`[
		{
			"name": "Average Duration",
			"description": "Create Release average duration",
			"type": "number",
			"number": {
				"unit": "seconds"
			},
			"rows": [
				{
					"repository_full_name": "flanksource/canary-checker",
					"value": "100"
				},
				{
					"repository_full_name": "flanksource/config-db",
					"value": "200"
				},
				{
					"repository_full_name": "flanksource/duty",
					"value": "300"
				}
			]
		}
	]`)),
}

var AllDummyViews = []models.View{
	View,
}

var AllDummyViewPanels = []models.ViewPanel{
	PipelineView,
}
