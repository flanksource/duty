package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPICanary = models.Canary{
	ID:        uuid.MustParse("0186b7a5-a2a4-86fd-c326-3a2104a2777f"),
	Name:      "dummy-logistics-api-canary",
	Namespace: "logistics",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
}

var LogisticsDBCanary = models.Canary{
	ID:        uuid.MustParse("0186b7a5-f246-3628-0d68-30bffc13244d"),
	Name:      "dummy-logistics-db-canary",
	Namespace: "logistics",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
}

var CartAPICanaryAgent = models.Canary{
	ID:        uuid.MustParse("6dc9d6dd-0b55-4801-837c-352d3abf9b70"),
	AgentID:   GCPAgent.ID,
	Name:      "dummy-cart-api-canary",
	Namespace: "cart",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
}

var AllDummyCanaries = []models.Canary{
	LogisticsAPICanary,
	LogisticsDBCanary,
	CartAPICanaryAgent,
}
