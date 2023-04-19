package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// Config Types
const (
	ConfigTypeCluster    = "Cluster"
	ConfigTypeDatabase   = "Database"
	ConfigTypeDeployment = "Deployment"
	ConfigTypeEC2        = "EC2"
	ConfigTypeNamespace  = "Namespace"
	ConfigTypeNode       = "Node"
	ConfigTypePod        = "Pod"
)

var EKSCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeCluster,
}

var KubernetesCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeCluster,
}

var KubernetesNodeA = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   ConfigTypeNode,
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   ConfigTypeNode,
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeEC2,
}

var EC2InstanceB = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeEC2,
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeDeployment,
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeDeployment,
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeDeployment,
}

var LogisticsDBRDS = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: ConfigTypeDatabase,
}

var AllDummyConfigs = []models.ConfigItem{
	EKSCluster,
	KubernetesCluster,
	KubernetesNodeA,
	KubernetesNodeB,
	EC2InstanceA,
	EC2InstanceB,
	LogisticsAPIDeployment,
	LogisticsUIDeployment,
	LogisticsWorkerDeployment,
	LogisticsDBRDS,
}
