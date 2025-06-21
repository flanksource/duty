package dummy

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
)

var EKSClusterCreateChange = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      EKSCluster.ID.String(),
	ChangeType:    "CREATE",
	CreatedAt:     &DummyYearOldDate,
	Severity:      models.SeverityMedium,
	Source:        "CloudTrail",
	Summary:       "EKS cluster created",
	Count:         1,
	FirstObserved: &DummyYearOldDate,
}

var EKSClusterUpdateChange = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      EKSCluster.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 24)),
	Severity:      models.SeverityLow,
	Source:        "CloudTrail",
	Summary:       "EKS cluster configuration updated",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 24)),
}

var EKSClusterDeleteChange = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      EKSCluster.ID.String(),
	ChangeType:    "DELETE",
	CreatedAt:     &DummyNow,
	Severity:      models.SeverityHigh,
	Source:        "CloudTrail",
	Summary:       "EKS cluster deleted",
	Count:         1,
	FirstObserved: &DummyNow,
}

var KubernetesNodeAChange = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      KubernetesNodeA.ID.String(),
	ChangeType:    "CREATE",
	CreatedAt:     &DummyYearOldDate,
	Severity:      models.SeverityInfo,
	Source:        "Kubernetes",
	Summary:       "Kubernetes node created",
	Count:         1,
	FirstObserved: &DummyYearOldDate,
}

var AllDummyConfigChanges = []models.ConfigChange{
	EKSClusterCreateChange,
	EKSClusterUpdateChange,
	EKSClusterDeleteChange,
	KubernetesNodeAChange,
}
