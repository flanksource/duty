package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSClusterCreateChange = models.ConfigChange{
	ID:         uuid.New().String(),
	ConfigID:   EKSCluster.ID.String(),
	ChangeType: "CREATE",
	CreatedAt:  &DummyYearOldDate,
}

var EKSClusterUpdateChange = models.ConfigChange{
	ID:         uuid.New().String(),
	ConfigID:   EKSCluster.ID.String(),
	ChangeType: "UPDATE",
}

var EKSClusterDeleteChange = models.ConfigChange{
	ID:         uuid.New().String(),
	ConfigID:   EKSCluster.ID.String(),
	ChangeType: "DELETE",
}

var KubernetesNodeAChange = models.ConfigChange{
	ID:         uuid.New().String(),
	ConfigID:   KubernetesNodeA.ID.String(),
	ChangeType: "CREATE",
}

var AllDummyConfigChanges = []models.ConfigChange{
	EKSClusterCreateChange,
	EKSClusterUpdateChange,
	EKSClusterDeleteChange,
	KubernetesNodeAChange,
}
