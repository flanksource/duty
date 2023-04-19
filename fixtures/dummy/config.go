package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTCluster,
}

var KubernetesCluster = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTCluster,
}

var KubernetesNodeA = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   models.CTNode,
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:           uuid.New(),
	ConfigType:   models.CTNode,
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTEC2,
}

var EC2InstanceB = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTEC2,
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTDeployment,
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTDeployment,
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTDeployment,
}

var LogisticsDBRDS = models.ConfigItem{
	ID:         uuid.New(),
	ConfigType: models.CTDatabase,
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
