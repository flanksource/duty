package dummy

import (
	"github.com/flanksource/duty/models"
)

var EKSClusterClusterComponentRelationship = models.ConfigComponentRelationship{
	ConfigID:    EKSCluster.ID,
	ComponentID: ClusterComponent.ID,
}

var KubernetesClusterClusterComponentRelationship = models.ConfigComponentRelationship{
	ConfigID:    KubernetesCluster.ID,
	ComponentID: ClusterComponent.ID,
}

var LogisticsDBRDSLogisticsDBComponentRelationship = models.ConfigComponentRelationship{
	ConfigID:    LogisticsDBRDS.ID,
	ComponentID: LogisticsDB.ID,
}

var EC2InstanceBNodeBRelationship = models.ConfigComponentRelationship{
	ConfigID:    EC2InstanceB.ID,
	ComponentID: NodeB.ID,
}

var AllDummyConfigComponentRelationships = []models.ConfigComponentRelationship{
	EKSClusterClusterComponentRelationship,
	KubernetesClusterClusterComponentRelationship,
	LogisticsDBRDSLogisticsDBComponentRelationship,
	EC2InstanceBNodeBRelationship,
}
