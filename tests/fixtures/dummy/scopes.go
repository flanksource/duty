package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var RLSTestUserMissionControlPodsScope = models.Scope{
	ID:          uuid.MustParse("a1b2c3d4-e5f6-4a5b-9c8d-7e6f5a4b3c2d"),
	Name:        "mission-control-pods",
	Namespace:   "missioncontrol",
	Description: "Scope for accessing pods in the missioncontrol namespace",
	Source:      models.SourceUI,
	CreatedAt:   DummyCreatedAt,
	UpdatedAt:   DummyCreatedAt,
	Targets: types.JSON(`[
    {
      "config": {
        "namespace": "missioncontrol",
        "tagSelector": "type=pod"
      }
    }
  ]`),
}

var AllDummyScopes = []models.Scope{
	RLSTestUserMissionControlPodsScope,
}
