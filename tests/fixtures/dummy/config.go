package dummy

import (
	"embed"
	"path/filepath"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/samber/lo"

	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
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
	ExternalID:  pq.StringArray{"aws/us-east-1/clusters", "production-eks"},
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

var KubernetesNodeA = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("node-a"),
	ConfigClass: models.ConfigClassNode,
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "node-a"}}`),
	Type:        lo.ToPtr("Kubernetes::Node"),
	ExternalID:  pq.StringArray{"aws/us-east-1/clusters", "kubernetes/nodes"},
	CreatedAt:   DummyCreatedAt.Add(time.Hour * 24),
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
	CostTotal30d: 50,
}

var KubernetesNodeB = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("node-b"),
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "node-b"}}`),
	ConfigClass: models.ConfigClassNode,
	Type:        lo.ToPtr("Kubernetes::Node"),
	ExternalID:  pq.StringArray{"aws/us-west-2/clusters", "kubernetes/nodes"},
	CreatedAt:   DummyCreatedAt.Add(time.Hour * 24 * 2),
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
	CostTotal30d: 80,
}

var EC2InstanceA = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassVirtualMachine,
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("EC2::Instance"),
	ExternalID:  pq.StringArray{"aws/us-east-1", "testing/instances"},
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
	ExternalID:  pq.StringArray{"aws/us-west-2", "production/instances"},
	Tags: types.JSONStringMap{
		"account": "flanksource",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"account":     "flanksource",
		"environment": "production",
		"app":         "frontend",
	}),
}

var LogisticsDBRDS = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassDatabase,
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Logistics::DB::RDS"),
	ExternalID:  pq.StringArray{"aws/us-east-1/rds", "logistics"},
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

var NginxHelmRelease = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("nginx-ingress"),
	ConfigClass: "HelmRelease",
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Helm::Release"),
	Status:      lo.ToPtr("deployed"),
	ExternalID:  pq.StringArray{"kubernetes/ingress-nginx", "helm/nginx"},
	Config: lo.ToPtr(`{
      "apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
      "kind": "HelmRelease",
      "metadata": {
        "name": "nginx-ingress",
        "namespace": "ingress-nginx"
      },
      "spec": {
        "chart": {
          "spec": {
            "chart": "ingress-nginx",
            "version": "4.8.0",
            "sourceRef": {
              "kind": "HelmRepository",
              "name": "ingress-nginx"
            }
          }
        },
        "interval": "5m",
        "values": {
          "controller": {
            "replicaCount": 2
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "nginx-ingress",
		"environment": "production",
		"owner":       "platform-team",
		"version":     "4.8.0",
		"chart":       "ingress-nginx",
	}),
	Tags: map[string]string{
		"namespace": "ingress-nginx",
		"chart":     "ingress-nginx",
		"release":   "nginx-ingress",
	},
}

var RedisHelmRelease = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("redis"),
	ConfigClass: "HelmRelease",
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Helm::Release"),
	Status:      lo.ToPtr("deployed"),
	ExternalID:  pq.StringArray{"kubernetes/database", "helm/redis"},
	Config: lo.ToPtr(`{
      "apiVersion": "helm.toolkit.fluxcd.io/v2beta1",
      "kind": "HelmRelease",
      "metadata": {
        "name": "redis",
        "namespace": "database"
      },
      "spec": {
        "chart": {
          "spec": {
            "chart": "redis",
            "version": "18.1.5",
            "sourceRef": {
              "kind": "HelmRepository",
              "name": "bitnami"
            }
          }
        },
        "interval": "10m",
        "values": {
          "replica": {
            "replicaCount": 1
          },
          "auth": {
            "enabled": true
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "redis",
		"environment": "production",
		"owner":       "data-team",
		"version":     "18.1.5",
		"chart":       "redis",
	}),
	Tags: map[string]string{
		"namespace": "database",
		"chart":     "redis",
		"release":   "redis",
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
	LogisticsUIReplicaSet,
	LogisticsUIPodConfig,
	LogisticsWorkerDeployment,
	LogisticsDBRDS,
	NginxHelmRelease,
	RedisHelmRelease,
}

var AzureConfigScraper = models.ConfigScraper{
	ID:     uuid.New(),
	Name:   "Azure scraper",
	Source: "ConfigFile",
	Spec:   "{}",
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

var AllConfigRelationships = []models.ConfigRelationship{
	ClusterAKSNodeRelationship,
	ClusterNodeARelationship,
	ClusterNodeBRelationship,
}

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
