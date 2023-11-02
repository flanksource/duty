package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsTopology = models.Topology{
	ID:        uuid.MustParse("df39086e-506b-4ad9-9af7-baf5275c382b"),
	Name:      "logistics",
	Namespace: "default",
}

var AllDummyTopologies = []models.Topology{
	LogisticsTopology,
}
