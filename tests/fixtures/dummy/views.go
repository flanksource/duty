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
	  "queries": {
		"pods": {
		  "configs": {
			"types": [
			  "Kubernetes::Pod"
			]
		  }
		}
	  },
	  "panels": [
		{
		  "name": "Pods",
		  "description": "Number of Pods",
		  "type": "gauge",
		  "gauge": {
			"min": "0",
			"max": "100",
			"thresholds": [
			  {
				"value": 0,
				"color": "green"
			  },
			  {
				"value": 60,
				"color": "orange"
			  },
			  {
				"value": 90,
				"color": "red"
			  }
			]
		  },
		  "query": "SELECT COUNT(*) AS value FROM pods"
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
