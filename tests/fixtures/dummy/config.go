package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

var EKSCluster = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("Production EKS"),
	ConfigClass: models.ConfigClassCluster,
	Type:        lo.ToPtr("EKS::Cluster"),
	Tags: types.JSONStringMap{
		"cluster": "aws",
		"account": "flanksource",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"account":     "flanksource",
		"cluster":     "aws",
		"environment": "production",
		"telemetry":   "enabled",
	}),
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        lo.ToPtr("Kubernetes::Cluster"),
	Tags: types.JSONStringMap{
		"cluster": "demo",
		"account": "flanksource",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"account":     "flanksource",
		"cluster":     "demo",
		"environment": "development",
		"telemetry":   "enabled",
	}),
}

var KubernetesNodeAKSPool1 = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("aks-pool-1"),
	ConfigClass: models.ConfigClassNode,
	Type:        lo.ToPtr("Kubernetes::Node"),
	Status:      lo.ToPtr("Healthy"),
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "aks-pool-1"}}`),
	Tags: types.JSONStringMap{
		"cluster":      "demo",
		"subscription": "018fbd67-bb86-90e1-07c9-243eedc73892",
	},
	Health: lo.ToPtr(models.HealthHealthy),
	Labels: lo.ToPtr(types.JSONStringMap{
		"cluster":      "demo",
		"subscription": "018fbd67-bb86-90e1-07c9-243eedc73892",
	}),
	Properties: &types.Properties{
		{Name: "memory", Value: lo.ToPtr(int64(64))},
	},
}

var KubernetesNodeA = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("node-a"),
	ConfigClass: models.ConfigClassNode,
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "node-a"}}`),
	Type:        lo.ToPtr("Kubernetes::Node"),
	Status:      lo.ToPtr("Healthy"),
	Tags: types.JSONStringMap{
		"cluster": "aws",
		"account": "flanksource",
	},
	Health: lo.ToPtr(models.HealthHealthy),
	Labels: lo.ToPtr(types.JSONStringMap{
		"cluster": "aws",
		"account": "flanksource",
		"role":    "worker",
		"region":  "us-east-1",
	}),
	Properties: &types.Properties{
		{Name: "memory", Value: lo.ToPtr(int64(64))},
		{Name: "region", Text: "us-east-1"},
	},
	CostTotal30d: 1,
}

var KubernetesNodeB = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("node-b"),
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "node-b"}}`),
	ConfigClass: models.ConfigClassNode,
	Type:        lo.ToPtr("Kubernetes::Node"),
	Status:      lo.ToPtr("Healthy"),
	Tags: types.JSONStringMap{
		"cluster": "aws",
		"account": "flanksource",
	},
	Health: lo.ToPtr(models.HealthHealthy),
	Labels: lo.ToPtr(types.JSONStringMap{
		"cluster":        "aws",
		"account":        "flanksource",
		"role":           "worker",
		"region":         "us-west-2",
		"storageprofile": "managed",
	}),
	Properties: &types.Properties{
		{Name: "memory", Value: lo.ToPtr(int64(32))},
		{Name: "region", Text: "us-west-2"},
	},
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        lo.ToPtr("EC2::Instance"),
	Tags: types.JSONStringMap{
		"account": "flanksource",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"account":     "flanksource",
		"environment": "testing",
		"app":         "backend",
	}),
}

var EC2InstanceB = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Type:        lo.ToPtr("EC2::Instance"),
	Tags: types.JSONStringMap{
		"account": "flanksource",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"account":     "flanksource",
		"environment": "production",
		"app":         "frontend",
	}),
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-1",
		"version":     "1.2.0",
	}),
}

var LogisticsAPIReplicaSet = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: "ReplicaSet",
	Type:        lo.ToPtr("Kubernetes::ReplicaSet"),
	ParentID:    lo.ToPtr(LogisticsAPIDeployment.ID),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-1",
		"version":     "1.2.0",
	}),
}

var LogisticsAPIPodConfig = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassPod,
	Type:        lo.ToPtr("Kubernetes::Pod"),
	ParentID:    lo.ToPtr(LogisticsAPIReplicaSet.ID),
	Labels: lo.ToPtr(types.JSONStringMap{
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
	Labels: lo.ToPtr(types.JSONStringMap{
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
	Labels: lo.ToPtr(types.JSONStringMap{
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
	Labels: lo.ToPtr(types.JSONStringMap{
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
	KubernetesNodeAKSPool1,
	EC2InstanceA,
	EC2InstanceB,
	LogisticsAPIDeployment,
	LogisticsAPIReplicaSet,
	LogisticsAPIPodConfig,
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
