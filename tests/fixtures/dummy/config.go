package dummy

import (
	"embed"
	"path/filepath"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

//go:embed config/*.yaml
var yamls embed.FS

func ImportConfigs(data []byte) (configs []models.ConfigItem, relationships []models.ConfigRelationship, err error) {
	objects, err := kubernetes.GetUnstructuredObjects(data)
	if err != nil {
		return nil, nil, err
	}

	for _, object := range objects {
		json, _ := object.MarshalJSON()
		labels := types.JSONStringMap{}
		for k, v := range object.GetLabels() {
			labels[k] = v
		}
		ci := models.ConfigItem{
			Config:      lo.ToPtr(string(json)),
			ID:          uuid.MustParse(string(object.GetUID())),
			Name:        lo.ToPtr(object.GetName()),
			ConfigClass: object.GetKind(),
			Type:        lo.ToPtr("Kubernetes::" + object.GetKind()),
			Labels:      lo.ToPtr(labels),
			CreatedAt:   object.GetCreationTimestamp().Time,
			Tags: types.JSONStringMap{
				"namespace": object.GetNamespace(),
			},
		}

		if parent, ok := object.GetAnnotations()["config-db.flanksource.com/parent"]; ok {
			id, err := uuid.Parse(parent)
			if err == nil {
				ci.ParentID = lo.ToPtr(id)
				relationships = append(relationships, models.ConfigRelationship{
					ConfigID:  id.String(),
					RelatedID: ci.ID.String(),
				})
			}
		}

		if related, ok := object.GetAnnotations()["config-db.flanksource.com/related"]; ok {
			for _, relation := range strings.Split(related, ",") {
				id, err := uuid.Parse(relation)
				if err == nil {
					relationships = append(relationships, models.ConfigRelationship{
						ConfigID:  ci.ID.String(),
						RelatedID: id.String(),
					})
				}

			}
		}
		configs = append(configs, ci)
	}
	return configs, relationships, nil
}

var EKSCluster = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("Production EKS"),
	ConfigClass: models.ConfigClassCluster,
	Health:      lo.ToPtr(models.HealthUnknown),
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
		"eks_version": "1.27",
	}),
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassCluster,
	Type:        lo.ToPtr("Kubernetes::Cluster"),
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Health:      lo.ToPtr(models.HealthUnknown),
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
	Status:      lo.ToPtr("healthy"),
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
	Status:      lo.ToPtr("healthy"),
	Tags: types.JSONStringMap{
		"cluster": "aws",
		"account": "flanksource",
		"region":  "us-east-1",
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
	Status:      lo.ToPtr("healthy"),
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
		{Name: "os", Text: "linux"},
	},
	CostTotal30d: 1.5,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Health:      lo.ToPtr(models.HealthHealthy),
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
	Health:      lo.ToPtr(models.HealthHealthy),
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
	Name:        lo.ToPtr("logistics-api"),
	Health:      lo.ToPtr(models.HealthHealthy),
	ConfigClass: models.ConfigClassDeployment,
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {
        "name": "logistics-api",
        "labels": {
          "app": "logistics-api"
        }
      },
      "spec": {
        "replicas": 3,
        "selector": {
          "matchLabels": {
            "app": "logistics-api"
          }
        },
        "template": {
          "metadata": {
            "labels": {
              "app": "logistics-api"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "logistics-api",
                "image": "logistics-api:latest",
                "ports": [
                  {
                    "containerPort": 80
                  }
                ]
              }
            ]
          }
        }
      }
    }`),
	Type: lo.ToPtr("Kubernetes::Deployment"),
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
	Name:        lo.ToPtr("logistics-api"),
	Type:        lo.ToPtr("Kubernetes::ReplicaSet"),
	Health:      lo.ToPtr(models.HealthHealthy),
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
	ParentID: lo.ToPtr(LogisticsAPIDeployment.ID),
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
	Name:        lo.ToPtr("logistics-api-pod-1"),
	Type:        lo.ToPtr("Kubernetes::Pod"),
	Health:      lo.ToPtr(models.HealthHealthy),
	Status:      lo.ToPtr("Running"),
	ParentID:    lo.ToPtr(LogisticsAPIReplicaSet.ID),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-1",
		"version":     "1.2.0",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("logistics-ui"),
	ConfigClass: models.ConfigClassDeployment,
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Logistics::UI::Deployment"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDeployment,
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Logistics::Worker::Deployment"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics",
		"environment": "production",
		"owner":       "team-3",
		"version":     "1.5.0",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
}

var LogisticsDBRDS = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDatabase,
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Logistics::DB::RDS"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"database":    "logistics",
		"environment": "production",
		"region":      "us-east-1",
		"size":        "large",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
	},
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

var KubeScrapeConfig = models.ConfigScraper{
	ID:        uuid.New(),
	Name:      "kubernetes-scraper",
	Namespace: "default",
	Source:    models.SourceUI,
	Spec: `{
    "kubernetes": [
      {
        "clusterName": "kubernetes",
        "kubeconfig": {
          "value": "/etc/my-kube-config"
        }
      }
    ]
  }`,
}

var AllConfigScrapers = []models.ConfigScraper{AzureConfigScraper, KubeScrapeConfig}

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

func GetConfig(configType, namespace, name string) models.ConfigItem {
	for _, config := range AllDummyConfigs {
		if *config.Type == configType &&
			*config.Name == name &&
			config.Tags["namespace"] == namespace {
			return config
		}
	}
	return models.ConfigItem{}
}

var GitRepository models.ConfigItem
var Kustomization models.ConfigItem
var Namespace models.ConfigItem

func init() {
	files, _ := yamls.ReadDir("config")
	for _, file := range files {
		data, err := yamls.ReadFile(filepath.Join("config", file.Name()))
		if err != nil {
			logger.Errorf("Failed to read %s: %v", file.Name(), err)
			continue
		}
		configs, relationships, err := ImportConfigs(data)
		if err != nil {
			logger.Errorf("Failed to import configs %v", err)
			continue
		}

		AllConfigRelationships = append(AllConfigRelationships, relationships...)
		AllDummyConfigs = append(AllDummyConfigs, configs...)
	}

	GitRepository = GetConfig("Kubernetes::GitRepository", "flux-system", "sandbox")
	Kustomization = GetConfig("Kubernetes::Kustomization", "flux-system", "infra")
	Namespace = GetConfig("Kubernetes::Namespace", "", "flux")

}
