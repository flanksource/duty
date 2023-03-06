package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPIHealthHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-0593-73e9-7e3d-5b3446336c1d"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-health-check",
	Type:     "http",
}

var LogisticsAPIHomeHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-625a-6a38-a9a7-e5e6b44ffec3"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-home-check",
	Type:     "http",
}

var LogisticsDBCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-9338-7142-1b10-25dc49030218"),
	CanaryID: LogisticsDBCanary.ID,
	Name:     "logistics-db-check",
	Type:     "postgres",
}

var AllDummyChecks = []models.Check{
	LogisticsAPIHealthHTTPCheck,
	LogisticsAPIHomeHTTPCheck,
	LogisticsDBCheck,
}
