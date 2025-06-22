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

// Nginx Helm Release version upgrade changes
var NginxHelmReleaseUpgradeV1 = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      NginxHelmRelease.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 72)), // 3 days ago
	Severity:      models.SeverityInfo,
	Source:        "Flux",
	Summary:       "Helm chart upgraded from 4.7.0 to 4.7.1",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 72)),
}

var NginxHelmReleaseUpgradeV2 = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      NginxHelmRelease.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 48)), // 2 days ago
	Severity:      models.SeverityInfo,
	Source:        "Flux",
	Summary:       "Helm chart upgraded from 4.7.1 to 4.7.2",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 48)),
}

var NginxHelmReleaseUpgradeV3 = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      NginxHelmRelease.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 24)), // 1 day ago
	Severity:      models.SeverityInfo,
	Source:        "Flux",
	Summary:       "Helm chart upgraded from 4.7.2 to 4.8.0",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 24)),
}

// Redis Helm Release version upgrade changes
var RedisHelmReleaseUpgradeV1 = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      RedisHelmRelease.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 96)), // 4 days ago
	Severity:      models.SeverityInfo,
	Source:        "Flux",
	Summary:       "Helm chart upgraded from 18.0.2 to 18.1.0",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 96)),
}

var RedisHelmReleaseUpgradeV2 = models.ConfigChange{
	ID:            uuid.New().String(),
	ConfigID:      RedisHelmRelease.ID.String(),
	ChangeType:    "UPDATE",
	CreatedAt:     lo.ToPtr(DummyNow.Add(-time.Hour * 60)), // 2.5 days ago
	Severity:      models.SeverityInfo,
	Source:        "Flux",
	Summary:       "Helm chart upgraded from 18.1.0 to 18.1.3",
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 60)),
}

var RedisHelmReleaseUpgradeV3 = models.ConfigChange{
	ID:         uuid.New().String(),
	ConfigID:   RedisHelmRelease.ID.String(),
	ChangeType: "UPDATE",
	CreatedAt:  lo.ToPtr(DummyNow.Add(-time.Hour * 36)), // 1.5 days ago
	Severity:   models.SeverityInfo,
	Source:     "Flux",
	Summary:    "Helm chart upgraded from 18.1.3 to 18.1.5",
	Details: []byte(`{
		"old_version": "18.1.3",
		"new_version": "18.1.5"
	}`),
	Count:         1,
	FirstObserved: lo.ToPtr(DummyNow.Add(-time.Hour * 36)),
}

var AllDummyConfigChanges = []models.ConfigChange{
	EKSClusterCreateChange,
	EKSClusterUpdateChange,
	EKSClusterDeleteChange,
	KubernetesNodeAChange,
	NginxHelmReleaseUpgradeV1,
	NginxHelmReleaseUpgradeV2,
	NginxHelmReleaseUpgradeV3,
	RedisHelmReleaseUpgradeV1,
	RedisHelmReleaseUpgradeV2,
	RedisHelmReleaseUpgradeV3,
}
