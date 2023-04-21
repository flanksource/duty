package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeCluster,
}

var KubernetesCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeCluster,
}

var KubernetesNodeA = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   models.ConfigTypeNode,
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   models.ConfigTypeNode,
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeVirtualMachine,
}

var EC2InstanceB = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeVirtualMachine,
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeDeployment,
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeDeployment,
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeDeployment,
}

var LogisticsDBRDS = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.ConfigTypeDatabase,
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
