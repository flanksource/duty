package dummy

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var KubeScrapeConfig = models.ConfigScraper{
	ID:        uuid.New(),
	Name:      "kubernetes-scraper",
	Namespace: "default",
	Source:    models.SourceUI,
	Spec: `{
    "kubernetes": [
      {
        "clusterName": "demo",
        "kubeconfig": {
          "value": "testdata/my-kube-config.yaml"
        }
      }
    ]
  }`,
}

var KubernetesCluster = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("demo"),
	ConfigClass: models.ConfigClassCluster,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Type:        lo.ToPtr("Kubernetes::Cluster"),
	Health:      lo.ToPtr(models.HealthUnknown),
	ExternalID: []string{
		"cluster://kubernetes/demo",
	},
	Tags: types.JSONStringMap{
		"cluster": "demo",
	},
	Labels: lo.ToPtr(types.JSONStringMap{
		"cluster":     "demo",
		"environment": "development",
		"telemetry":   "enabled",
	}),
}

var MissionControlNamespace = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("missioncontrol"),
	Type:        lo.ToPtr("Kubernetes::Namespace"),
	ConfigClass: models.ConfigClassNamespace,
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Namespace", "metadata": {"name": "missioncontrol"}}`),
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	ExternalID: []string{
		"namespace://kubernetes/demo/missioncontrol",
	},
}

var KubernetesNodeAKSPool1 = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("aks-pool-1"),
	ConfigClass: models.ConfigClassNode,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Type:        lo.ToPtr("Kubernetes::Node"),
	CreatedAt:   DummyCreatedAt,
	Status:      lo.ToPtr("healthy"),
	Config:      lo.ToPtr(`{"apiVersion":"v1", "kind":"Node", "metadata": {"name": "aks-pool-1"}}`),
	ExternalID: []string{
		"node://kubernetes/aks-pool-1",
	},
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
	CostTotal30d: 100,
}

var LogisticsAPIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("logistics-api"),
	Health:      lo.ToPtr(models.HealthHealthy),
	ConfigClass: models.ConfigClassDeployment,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	ExternalID: []string{
		"deployment://kubernetes/demo/missioncontrol/logistics-api",
	},
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {
        "name": "logistics-api",
        "namespace": "missioncontrol",
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
                ],
                "resources": {
                  "requests": {
                    "memory": "128Mi",
                    "cpu": "100m"
                  },
                  "limits": {
                    "memory": "256Mi",
                    "cpu": "500m"
                  }
                }
              }
            ]
          }
        }
      }
    }`),
	Type: lo.ToPtr("Kubernetes::Deployment"),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app": "logistics-api",
	}),
}

var LogisticsAPIReplicaSet = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: "ReplicaSet",
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Name:        lo.ToPtr("logistics-api-7df4c7f6b7"),
	Type:        lo.ToPtr("Kubernetes::ReplicaSet"),
	Health:      lo.ToPtr(models.HealthHealthy),
	ExternalID: []string{
		"replicaset://kubernetes/demo/missioncontrol/logistics-api-7df4c7f6b7",
	},
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
	ParentID: lo.ToPtr(LogisticsAPIDeployment.ID),
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "ReplicaSet",
      "metadata": {
        "name": "logistics-api-7df4c7f6b7",
        "namespace": "missioncontrol",
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
                ],
                "resources": {
                  "requests": {
                    "memory": "128Mi",
                    "cpu": "100m"
                  },
                  "limits": {
                    "memory": "256Mi",
                    "cpu": "500m"
                  }
                }
              }
            ]
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app": "logistics-api",
	}),
}

var LogisticsAPIPodConfig = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassPod,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Name:        lo.ToPtr("logistics-api-7df4c7f6b7-x9k2m"),
	Type:        lo.ToPtr("Kubernetes::Pod"),
	Health:      lo.ToPtr(models.HealthHealthy),
	CreatedAt:   DummyCreatedAt,
	Status:      lo.ToPtr("Running"),
	ExternalID: []string{
		"pod://kubernetes/demo/missioncontrol/logistics-api-7df4c7f6b7-x9k2m",
	},
	ParentID: lo.ToPtr(LogisticsAPIReplicaSet.ID),
	Config: lo.ToPtr(`{
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "logistics-api-7df4c7f6b7-x9k2m",
        "namespace": "missioncontrol",
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
            ],
            "resources": {
              "requests": {
                "memory": "128Mi",
                "cpu": "100m"
              },
              "limits": {
                "memory": "256Mi",
                "cpu": "500m"
              }
            }
          }
        ]
      },
      "status": {
        "phase": "Running"
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app": "logistics-api",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
	CostTotal30d: 5,
}

var LogisticsUIDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("logistics-ui"),
	ConfigClass: models.ConfigClassDeployment,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {
        "name": "logistics-ui",
        "namespace": "missioncontrol",
        "labels": {
          "app": "logistics-ui",
          "owner": "team-2"
        }
      },
      "spec": {
        "replicas": 1,
        "selector": {
          "matchLabels": {
            "app": "logistics-ui"
          }
        },
        "template": {
          "metadata": {
            "labels": {
              "app": "logistics-ui",
              "owner": "team-2",
              "environment": "production",
              "version": "2.0.1"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "logistics-ui",
                "image": "logistics-ui:2.0.1",
                "ports": [
                  {
                    "containerPort": 8080
                  }
                ],
                "env": [
                  {
                    "name": "API_ENDPOINT",
                    "value": "http://logistics-api"
                  }
                ],
                "resources": {
                  "requests": {
                    "memory": "64Mi",
                    "cpu": "50m"
                  },
                  "limits": {
                    "memory": "128Mi",
                    "cpu": "250m"
                  }
                }
              }
            ]
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics-ui",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
}

var LogisticsUIReplicaSet = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: "ReplicaSet",
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Name:        lo.ToPtr("logistics-ui-6c8f9b4d5e"),
	Type:        lo.ToPtr("Kubernetes::ReplicaSet"),
	Health:      lo.ToPtr(models.HealthHealthy),
	ExternalID: []string{
		"replicaset://kubernetes/demo/missioncontrol/logistics-ui-6c8f9b4d5e",
	},
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
	ParentID: lo.ToPtr(LogisticsUIDeployment.ID),
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "ReplicaSet",
      "metadata": {
        "name": "logistics-ui-6c8f9b4d5e",
        "namespace": "missioncontrol",
        "labels": {
          "app": "logistics-ui",
          "owner": "team-2"
        }
      },
      "spec": {
        "replicas": 1,
        "selector": {
          "matchLabels": {
            "app": "logistics-ui"
          }
        },
        "template": {
          "metadata": {
            "labels": {
              "app": "logistics-ui",
              "owner": "team-2",
              "environment": "production",
              "version": "2.0.1"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "logistics-ui",
                "image": "logistics-ui:2.0.1",
                "ports": [
                  {
                    "containerPort": 8080
                  }
                ],
                "env": [
                  {
                    "name": "API_ENDPOINT",
                    "value": "http://logistics-api"
                  }
                ],
                "resources": {
                  "requests": {
                    "memory": "64Mi",
                    "cpu": "50m"
                  },
                  "limits": {
                    "memory": "128Mi",
                    "cpu": "250m"
                  }
                }
              }
            ]
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics-ui",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
}

var LogisticsUIPodConfig = models.ConfigItem{
	ID:          uuid.New(),
	ConfigClass: models.ConfigClassPod,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Name:        lo.ToPtr("logistics-ui-6c8f9b4d5e-m7n8p"),
	Type:        lo.ToPtr("Kubernetes::Pod"),
	Health:      lo.ToPtr(models.HealthHealthy),
	CreatedAt:   DummyCreatedAt,
	Status:      lo.ToPtr("Running"),
	ParentID:    lo.ToPtr(LogisticsUIReplicaSet.ID),
	Config: lo.ToPtr(`{
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "logistics-ui-6c8f9b4d5e-m7n8p",
        "namespace": "missioncontrol",
        "labels": {
          "app": "logistics-ui",
          "owner": "team-2"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "logistics-ui",
            "image": "logistics-ui:2.0.1",
            "ports": [
              {
                "containerPort": 8080
              }
            ],
            "env": [
              {
                "name": "API_ENDPOINT",
                "value": "http://logistics-api"
              }
            ],
            "resources": {
              "requests": {
                "memory": "64Mi",
                "cpu": "50m"
              },
              "limits": {
                "memory": "128Mi",
                "cpu": "250m"
              }
            }
          }
        ]
      },
      "status": {
        "phase": "Running"
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics-ui",
		"environment": "production",
		"owner":       "team-2",
		"version":     "2.0.1",
	}),
	Tags: map[string]string{
		"cluster":   "demo",
		"namespace": "missioncontrol",
	},
}

var LogisticsWorkerDeployment = models.ConfigItem{
	ID:          uuid.New(),
	Name:        lo.ToPtr("logistics-worker"),
	ConfigClass: models.ConfigClassDeployment,
	ScraperID:   lo.ToPtr(KubeScrapeConfig.ID.String()),
	Health:      lo.ToPtr(models.HealthHealthy),
	Type:        lo.ToPtr("Kubernetes::Deployment"),
	Config: lo.ToPtr(`{
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {
        "name": "logistics-worker",
        "namespace": "missioncontrol",
        "labels": {
          "app": "logistics-worker",
          "owner": "team-3"
        }
      },
      "spec": {
        "replicas": 1,
        "selector": {
          "matchLabels": {
            "app": "logistics-worker"
          }
        },
        "template": {
          "metadata": {
            "labels": {
              "app": "logistics-worker"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "logistics-worker",
                "image": "logistics-worker:1.5.0",
                "env": [
                  {
                    "name": "QUEUE_URL",
                    "value": "redis://redis:6379"
                  },
                  {
                    "name": "DATABASE_URL",
                    "value": "postgres://logistics-db:5432/logistics"
                  }
                ],
                "resources": {
                  "requests": {
                    "memory": "256Mi",
                    "cpu": "100m"
                  },
                  "limits": {
                    "memory": "512Mi",
                    "cpu": "500m"
                  }
                }
              }
            ]
          }
        }
      }
    }`),
	Labels: lo.ToPtr(types.JSONStringMap{
		"app":         "logistics-worker",
		"environment": "production",
		"owner":       "team-3",
		"version":     "1.5.0",
	}),
	Tags: map[string]string{
		"namespace": "missioncontrol",
		"cluster":   "demo",
	},
}

var ClusterAKSNodeRelationship = models.ConfigRelationship{
	ConfigID:  KubernetesCluster.ID.String(),
	RelatedID: KubernetesNodeAKSPool1.ID.String(),
	Relation:  "ClusterNode",
}
