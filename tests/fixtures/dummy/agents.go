package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

var GCPAgent = models.Agent{
	ID:   uuid.MustParse("ebd4cbf7-267e-48f9-a050-eca12e535ce1"),
	Name: "GCP",
}

var HomelabAgent = models.Agent{
	ID:   uuid.MustParse("ac4b1dc5-b249-471d-89d7-ba0c5de4997b"),
	Name: "homelab",
}

var AllDummyAgents = []models.Agent{
	GCPAgent,
	HomelabAgent,
}
