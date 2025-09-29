package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

var EchoConfig = models.Playbook{
	ID:          uuid.MustParse("07ffd27a-b33f-4ee6-80d6-b83430a4a16e"),
	Name:        "echo-config",
	Namespace:   "default",
	Title:       "Echo config",
	Description: "echos the config spec",
	Source:      models.SourceUI,
	Category:    "debug",
	Spec:        []byte("{}"),
}

var AllDummyPlaybooks = []models.Playbook{EchoConfig}
