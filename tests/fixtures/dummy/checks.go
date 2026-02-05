package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

var LogisticsAPIHealthHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-0593-73e9-7e3d-5b3446336c1d"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-health-check",
	Type:     "http",
	Status:   "healthy",
	Labels: map[string]string{
		"app":       "logistics",
		"cluster":   "production-us",
		"namespace": "logistics",
		"env":       "production",
		"region":    "us-east-1",
		"pod":       "logistics-api-7b9d4f5c6-x2k4m",
		"pod_hash":  "7b9d4f5c6",
	},
}

var LogisticsAPIHomeHTTPCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-625a-6a38-a9a7-e5e6b44ffec3"),
	CanaryID: LogisticsAPICanary.ID,
	Name:     "logistics-api-home-check",
	Type:     "http",
	Status:   "healthy",
	Labels: map[string]string{
		"app":       "logistics",
		"cluster":   "production-us",
		"namespace": "logistics",
		"env":       "production",
		"instance":  "i-0abc123def456",
		"revision":  "12345",
	},
}

var LogisticsDBCheck = models.Check{
	ID:       uuid.MustParse("0186b7a4-9338-7142-1b10-25dc49030218"),
	CanaryID: LogisticsDBCanary.ID,
	Name:     "logistics-db-check",
	Type:     "postgres",
	Status:   "unhealthy",
	Labels: map[string]string{
		"app":       "logistics",
		"cluster":   "staging-eu",
		"namespace": "logistics",
		"env":       "staging",
		"region":    "eu-west-1",
	},
}

var CartAPIHeathCheckAgent = models.Check{
	ID:       uuid.MustParse("eed7bd6e-529b-4693-aca9-43977bcc5ff1"),
	AgentID:  GCPAgent.ID,
	CanaryID: CartAPICanaryAgent.ID,
	Name:     "cart-api-health-check",
	Type:     "http",
	Status:   models.CheckHealthStatus(types.ComponentStatusHealthy),
}

var DeletedCheck, DeletedCheck1h, DeletedCheckOld models.Check

func AllDummyChecks() []models.Check {
	DeletedCheck = models.Check{
		ID:        uuid.MustParse("eed7bd6e-529b-4693-aca9-55177bcc5ff1"),
		AgentID:   GCPAgent.ID,
		CanaryID:  CartAPICanaryAgent.ID,
		CreatedAt: lo.ToPtr(CurrentTime.Add(-10 * time.Minute)),
		DeletedAt: lo.ToPtr(CurrentTime.Add(-5 * time.Minute)),
		Name:      "cart-deleted-5m-ago",
		Type:      "http",
		Status:    models.CheckHealthStatus(types.ComponentStatusHealthy),
	}
	DeletedCheck1h = models.Check{
		ID:        uuid.MustParse("eed7bd6e-529b-4693-aca9-55177bcc5ff2"),
		AgentID:   GCPAgent.ID,
		CanaryID:  CartAPICanaryAgent.ID,
		CreatedAt: lo.ToPtr(CurrentTime.Add(-120 * time.Minute)),
		DeletedAt: lo.ToPtr(CurrentTime.Add(-100 * time.Minute)),
		Name:      "cart-deleted-2h-ago",
		Type:      "http",
		Status:    models.CheckHealthStatus(types.ComponentStatusHealthy),
	}
	DeletedCheckOld = models.Check{
		ID:        uuid.MustParse("eed8bd6e-529b-4693-aca9-55177bcc5ff1"),
		AgentID:   GCPAgent.ID,
		CanaryID:  CartAPICanaryAgent.ID,
		CreatedAt: lo.ToPtr(CurrentTime.Add(-1000 * time.Hour)),
		DeletedAt: lo.ToPtr(CurrentTime.Add(-999 * time.Hour)),
		Name:      "cart-deleted-41h-ago",
		Type:      "http",
		Status:    models.CheckHealthStatus(types.ComponentStatusHealthy),
	}
	return []models.Check{
		DeletedCheck, DeletedCheckOld, DeletedCheck1h,
		LogisticsAPIHealthHTTPCheck,
		LogisticsAPIHomeHTTPCheck,
		LogisticsDBCheck,
		CartAPIHeathCheckAgent,
	}
}
