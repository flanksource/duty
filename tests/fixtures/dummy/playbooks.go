package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EchoConfig = models.Playbook{
	ID:          uuid.MustParse("07ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	Name:        "echo-config",
	Namespace:   "default",
	Title:       "Echo config",
	Description: "echos the config spec",
	Source:      models.SourceUI,
	Category:    "debug",
	Tags: map[string]string{
		"category": "debug",
	},
	Spec: []byte("{}"),
}

var AllDummyPlaybooks = []models.Playbook{EchoConfig}
