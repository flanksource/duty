package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPICanary = models.Canary{
	ID:        uuid.New(),
	Name:      "dummy-logistics-api-canary",
	Namespace: "logistics",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
}

var LogisticsDBCanary = models.Canary{
	ID:        uuid.New(),
	Name:      "dummy-logistics-db-canary",
	Namespace: "logistics",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
}

var AllDummyCanaries = []models.Canary{
	LogisticsAPICanary,
	LogisticsDBCanary,
}
