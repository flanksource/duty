package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

var LogisticsAPIHealthHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-0593-73e9-7e3d-5b3446336c1d"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-health-check",
	Type:     "http",
	Status:   "healthy",
}

var LogisticsAPIHomeHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-625a-6a38-a9a7-e5e6b44ffec3"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-home-check",
	Type:     "http",
	Status:   "healthy",
}

var LogisticsDBCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-9338-7142-1b10-25dc49030218"),
	CanaryID: LogisticsDBCanary.ID,
	Name:     "logistics-db-check",
	Type:     "postgres",
	Status:   "unhealthy",
}

var CartAPIHeathCheckAgent = models.Check{
	ID:       uuid.MustParse("eed7bd6e-529b-4693-aca9-43977bcc5ff1"),
	AgentID:  &GCPAgent.ID,
	CanaryID: CartAPICanaryAgent.ID,
	Name:     "cart-api-health-check",
	Type:     "http",
	Status:   models.CheckHealthStatus(types.ComponentStatusHealthy),
}

var AllDummyChecks = []models.Check{
	LogisticsAPIHealthHTTPCheck,
	LogisticsAPIHomeHTTPCheck,
	LogisticsDBCheck,
	CartAPIHeathCheckAgent,
}
