package dummy

import (
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSClusterCreateChange = models.ConfigChange{
	ID:               uuid.New().String(),
	ConfigID:         EKSCluster.ID.String(),
	ChangeType:       "CREATE",
	ExternalChangeId: utils.RandomString(10),
}

var EKSClusterDeleteChange = models.ConfigChange{
	ID:               uuid.New().String(),
	ConfigID:         EKSCluster.ID.String(),
	ChangeType:       "DELETE",
	ExternalChangeId: utils.RandomString(10),
}

var KubernetesNodeAChange = models.ConfigChange{
	ID:               uuid.New().String(),
	ConfigID:         KubernetesNodeA.ID.String(),
	ChangeType:       "CREATE",
	ExternalChangeId: utils.RandomString(10),
}

var AllDummyConfigChanges = []models.ConfigChange{
	EKSClusterCreateChange,
	EKSClusterDeleteChange,
	KubernetesNodeAChange,
}
