package dummy

import (
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

var EKSCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        utils.Ptr("EKS::Cluster"),
	Tags: utils.Ptr(types.JSONStringMap{
		"telemetry":   "enabled",
		"environment": "production",
	}),
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        utils.Ptr("Kubernetes::Cluster"),
	Tags: utils.Ptr(types.JSONStringMap{
		"telemetry":   "enabled",
		"environment": "development",
	}),
}

var KubernetesNodeA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassNode,
	Type:        utils.Ptr("Kubernetes::Node"),
	Tags: utils.Ptr(types.JSONStringMap{
		"role":   "worker",
		"region": "us-east-1",
	}),
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassNode,
	Type:        utils.Ptr("Kubernetes::Node"),
	Tags: utils.Ptr(types.JSONStringMap{
		"role":           "worker",
		"region":         "us-west-2",
		"storageprofile": "managed",
	}),
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        utils.Ptr("EC2::Instance"),
	Tags: utils.Ptr(types.JSONStringMap{
		"environment": "testing",
		"app":         "backend",
	}),
}

var EC2InstanceB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        utils.Ptr("EC2::Instance"),
	Tags: utils.Ptr(types.JSONStringMap{
		"environment": "production",
		"app":         "frontend",
	}),
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        utils.Ptr("Logistics::API::Deployment"),
	Tags: utils.Ptr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-1",
		"version":     "1.2.0",
	}),
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        utils.Ptr("Logistics::UI::Deployment"),
	Tags: utils.Ptr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        utils.Ptr("Logistics::Worker::Deployment"),
	Tags: utils.Ptr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-3",
		"version":     "1.5.0",
	}),
}

var LogisticsDBRDS = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDatabase,
	Type:        utils.Ptr("Logistics::DB::RDS"),
	Tags: utils.Ptr(types.JSONStringMap{
		"database":    "logistics",
		"environment": "production",
		"region":      "us-east-1",
		"size":        "large",
	}),
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

var AzureConfigScraper = models.ConfigScraper{
	ID:     uuid.New(),
	Name:   "Azure scraper",
	Source: "ConfigFile",
	Spec:   "{}",
}

var AllConfigScrapers = []models.ConfigScraper{AzureConfigScraper}
