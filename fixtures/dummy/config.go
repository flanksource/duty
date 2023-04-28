package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
}

var KubernetesNodeA = models.ConfigItem{
	ID:           uuid.New(),
	ConfigClass:  models.ConfigClassNode,
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:           uuid.New(),
	ConfigClass:  models.ConfigClassNode,
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
}

var EC2InstanceB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
}

var LogisticsDBRDS = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDatabase,
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
