package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID: uuid.New(),
}

var KubernetesCluster = models.ConfigItem{
	ID: uuid.New(),
}

var KubernetesNodeA = models.ConfigItem{
	ID: uuid.New(),
}

var KubernetesNodeB = models.ConfigItem{
	ID: uuid.New(),
}

var EC2InstanceA = models.ConfigItem{
	ID: uuid.New(),
}

var EC2InstanceB = models.ConfigItem{
	ID: uuid.New(),
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID: uuid.New(),
}

var LogisticsUIDeployment = models.ConfigItem{
	ID: uuid.New(),
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID: uuid.New(),
}

var LogisticsDBRDS = models.ConfigItem{
	ID: uuid.New(),
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
