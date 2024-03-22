package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

var EKSCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        lo.ToPtr("EKS::Cluster"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"telemetry":   "enabled",
		"environment": "production",
	}),
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        lo.ToPtr("Kubernetes::Cluster"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"telemetry":   "enabled",
		"environment": "development",
	}),
}

var KubernetesNodeA = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("node-a"),
	ConfigClass: models.ConfigClassNode,
	Type:        lo.ToPtr("Kubernetes::Node"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"role":   "worker",
		"region": "us-east-1",
	}),
	Properties: &types.Properties{
		{Name: "memory", Value: 64},
		{Name: "region", Text: "eu-west-1"},
	},
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassNode,
	Type:        lo.ToPtr("Kubernetes::Node"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"role":           "worker",
		"region":         "us-west-2",
		"storageprofile": "managed",
	}),
	Properties: &types.Properties{
		{Name: "memory", Value: 32},
		{Name: "region", Text: "eu-west-2"},
	},
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        lo.ToPtr("EC2::Instance"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"environment": "testing",
		"app":         "backend",
	}),
}

var EC2InstanceB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        lo.ToPtr("EC2::Instance"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"environment": "production",
		"app":         "frontend",
	}),
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        lo.ToPtr("Logistics::API::Deployment"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-1",
		"version":     "1.2.0",
	}),
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        lo.ToPtr("Logistics::UI::Deployment"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        lo.ToPtr("Logistics::Worker::Deployment"),
	Tags: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-3",
		"version":     "1.5.0",
	}),
}

var LogisticsDBRDS = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDatabase,
	Type:        lo.ToPtr("Logistics::DB::RDS"),
	Tags: lo.ToPtr(types.JSONStringMap{
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

var ClusterNodeARelationship = models.ConfigRelationship{
	ConfigID:  KubernetesCluster.ID.String(),
	RelatedID: KubernetesNodeA.ID.String(),
	Relation:  "ClusterNode",
}

var ClusterNodeBRelationship = models.ConfigRelationship{
	ConfigID:  KubernetesCluster.ID.String(),
	RelatedID: KubernetesNodeB.ID.String(),
	Relation:  "ClusterNode",
}

var AllConfigRelationships = []models.ConfigRelationship{ClusterNodeARelationship, ClusterNodeBRelationship}
