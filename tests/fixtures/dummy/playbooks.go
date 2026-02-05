package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

var EchoConfig = models.Playbook{
	ID:          uuid.MustParse("07ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	Name:        "echo-config",
	Namespace:   "mc",
	Title:       "Echo config",
	Description: "Echos the name of the pod",
	Source:      models.SourceUI,
	Category:    "Echoer",
	Spec: []byte(`{
		"category": "Echoer",
		"description": "Echos the name of the pod",
		"configs": [
			{
				"name": "*"
			}
		],
		"actions": [
			{
				"name": "Echo name & agent",
				"exec": {
					"script": "echo \"Name: {{.config.name}} Agent: {{.agent.name}}\""
				}
			}
		]
	}`),
}

var RestartPod = models.Playbook{
	ID:          uuid.MustParse("17ffd27a-b33f-4ee6-80d6-b83430a4a16f"),
	Name:        "restart-pod",
	Namespace:   "mc",
	Title:       "Restart Pod",
	Description: "Restarts a Kubernetes pod",
	Source:      models.SourceUI,
	Category:    "Kubernetes",
	Spec: []byte(`{
		"category": "Kubernetes",
		"description": "Restarts a Kubernetes pod",
		"configs": [
			{
				"type": "Kubernetes::Pod"
			}
		],
		"actions": [
			{
				"name": "Delete pod",
				"exec": {
					"script": "kubectl delete pod {{.config.name}} -n {{.config.tags.namespace}}"
				}
			}
		]
	}`),
}

var AllDummyPlaybooks = []models.Playbook{EchoConfig, RestartPod}
