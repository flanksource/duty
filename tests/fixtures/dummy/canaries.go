package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
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

var UICanary = models.Canary{
	ID:        uuid.MustParse("c69f14cd-0041-4012-89f8-b5ed446ed8e9"),
	Name:      "ui-canary",
	Namespace: "cart",
	Spec:      []byte("{}"),
	CreatedAt: DummyCreatedAt,
	DeletedAt: lo.ToPtr(DummyCreatedAt.Add(time.Hour)),
}

var AllDummyCanaries = []models.Canary{
	LogisticsAPICanary,
	LogisticsDBCanary,
	CartAPICanaryAgent,
	UICanary,
}
