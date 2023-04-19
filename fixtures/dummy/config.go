package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "kubernetesCluster",
}

var KubernetesCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "kubernetesCluster",
}

var KubernetesNodeA = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   "kubernetesNode",
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   "kubernetesNode",
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "ec2",
}

var EC2InstanceB = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "ec2",
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "deployment",
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "deployment",
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "deployment",
}

var LogisticsDBRDS = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: "database",
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
