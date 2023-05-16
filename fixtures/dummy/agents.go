package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var GCPAgent = models.Agent{
	ID:   uuid.MustParse("ebd4cbf7-267e-48f9-a050-eca12e535ce1"),
	Name: "GCP",
}

var AllDummyAgents = []models.Agent{
	GCPAgent,
}
