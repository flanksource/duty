package dummy

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

//go:embed views/*.yaml
var viewYamls embed.FS

func ImportViews(data []byte) ([]models.View, error) {
	objects, err := kubernetes.GetUnstructuredObjects(data)
	if err != nil {
		return nil, err
	}

	var views []models.View
	for _, object := range objects {
		// Extract the spec from the CRD
		spec, found, err := unstructured.NestedMap(object.Object, "spec")
		if err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("spec not found: %s", object.GetName())
		}

		specJSON, err := json.Marshal(spec)
		if err != nil {
			return nil, err
		}

		view := models.View{
			ID:        uuid.MustParse(string(object.GetUID())),
			Name:      object.GetName(),
			Namespace: object.GetNamespace(),
			Spec:      types.JSON(specJSON),
			Source:    models.SourceCRD,
			CreatedAt: object.GetCreationTimestamp().Time,
		}

		ImportedDummyViews[view.Namespace+"/"+view.Name] = view
		views = append(views, view)
	}

	return views, nil
}

var PodView = models.View{
	ID:        uuid.New(),
	Name:      "pods",
	Namespace: "default",
	Labels: types.JSONStringMap{
		"environment": "production",
		"team":        "platform",
		"version":     "v1.2.0",
	},
	Spec: types.JSON([]byte(`{
   "display": {
      "title": "Pods",
      "icon": "pod"
   },
   "queries": {
      "pods": {
         "configs": {
            "types": [
               "Kubernetes::Pod"
            ]
         }
      }
   },
   "panels": [
      {
         "name": "Pods",
         "description": "Number of Pods",
         "type": "gauge",
         "gauge": {
            "min": "0",
            "max": "100",
            "thresholds": [
               {
                  "value": 0,
                  "color": "green"
               },
               {
                  "value": 60,
                  "color": "orange"
               },
               {
                  "value": 90,
                  "color": "red"
               }
            ]
         },
         "query": "SELECT COUNT(*) AS value FROM pods"
      }
   ],
   "columns": [
      {
         "name": "id",
         "type": "string",
         "primaryKey": true
      },
      {
         "name": "name",
         "type": "string"
      },
      {
         "name": "status",
         "type": "status"
      }
   ]
}`)),
	Source:    "KubernetesCRD",
	CreatedBy: lo.ToPtr(JohnDoe.ID),
	CreatedAt: DummyCreatedAt,
}

var ViewDev = models.View{
	ID:        uuid.New(),
	Name:      "Dev Dashboard",
	Namespace: "development",
	Labels: types.JSONStringMap{
		"environment": "development",
		"team":        "platform",
		"version":     "v1.1.0",
	},
	Spec: types.JSON([]byte(`{
	  "queries": {
			"services": {
				"configs": {
					"types": [
						"Kubernetes::Service"
					]
				}
			}
		},
	  "panels": [
		{
		  "name": "Services",
		  "description": "Number of Services",
		  "type": "stat",
		  "query": "SELECT COUNT(*) AS value FROM services"
		}
	  ]
	}`)),
	Source:    "KubernetesCRD",
	CreatedBy: lo.ToPtr(JohnDoe.ID),
	CreatedAt: DummyCreatedAt,
}

// Populated by ImportViews in init()
var (
	ImportedDummyViews = map[string]models.View{}
)

var PipelineView = models.ViewPanel{
	ViewID:   PodView.ID,
	AgentID:  uuid.Nil,
	IsPushed: false,
	Results: types.JSON([]byte(`[
		{
			"name": "Average Duration",
			"description": "Create Release average duration",
			"type": "number",
			"number": {
				"unit": "seconds"
			},
			"rows": [
				{
					"repository_full_name": "flanksource/canary-checker",
					"value": "100"
				},
				{
					"repository_full_name": "flanksource/config-db",
					"value": "200"
				},
				{
					"repository_full_name": "flanksource/duty",
					"value": "300"
				}
			]
		}
	]`)),
}

var AllDummyViews = []models.View{
	PodView,
	ViewDev,
}

var DevViewPanel = models.ViewPanel{
	ViewID:   ViewDev.ID,
	AgentID:  uuid.Nil,
	IsPushed: false,
	Results: types.JSON([]byte(`[
		{
			"name": "Service Status",
			"description": "Development services status overview",
			"type": "stat",
			"stat": {
				"unit": "services"
			},
			"rows": [
				{
					"namespace": "development",
					"status": "healthy",
					"value": "42"
				},
				{
					"namespace": "development",
					"status": "warning",
					"value": "5"
				},
				{
					"namespace": "development",
					"status": "error",
					"value": "2"
				}
			]
		}
	]`)),
}

var AllDummyViewPanels = []models.ViewPanel{
	PipelineView,
	DevViewPanel,
}

type ViewGeneratedTable struct {
	View models.View
	Rows []map[string]any
}

var PodViewTable = ViewGeneratedTable{
	View: PodView,
	Rows: []map[string]any{
		{
			"id":     NginxIngressPod.ID.String(),
			"name":   *NginxIngressPod.Name,
			"status": *NginxIngressPod.Status,
		},
		{
			"id":     LogisticsAPIPodConfig.ID.String(),
			"name":   *LogisticsAPIPodConfig.Name,
			"status": *LogisticsAPIPodConfig.Status,
		},
		{
			"id":     LogisticsUIPodConfig.ID.String(),
			"name":   *LogisticsUIPodConfig.Name,
			"status": *LogisticsUIPodConfig.Status,
		},
	},
}

var AllDummyViewTables = []ViewGeneratedTable{
	PodViewTable,
}

func init() {
	files, err := viewYamls.ReadDir("views")
	if err != nil {
		logger.Errorf("Failed to read embedded views: %v", err)
		os.Exit(1)
	}

	for _, file := range files {
		data, err := viewYamls.ReadFile(filepath.Join("views", file.Name()))
		if err != nil {
			logger.Errorf("Failed to read %s: %v", file.Name(), err)
			os.Exit(1)
		}

		views, err := ImportViews(data)
		if err != nil {
			logger.Errorf("Failed to import views %v", err)
			os.Exit(1)
		}

		AllDummyViews = append(AllDummyViews, views...)
	}
}
