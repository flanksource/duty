package dummy

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

var Logistics = models.Component{
	ID:         uuid.MustParse("018681fc-e54f-bd4f-42be-068a9a69eeb5"),
	Name:       "logistics",
	Type:       "Entity",
	ExternalId: "dummy/logistics",
	Labels:     types.JSONStringMap{"telemetry": "enabled"},
	Owner:      "logistics-team",
	CreatedAt:  DummyCreatedAt,
	Status:     types.ComponentStatusHealthy,
	Health:     lo.ToPtr(models.HealthHealthy),
}

var LogisticsAPI = models.Component{
	ID:         uuid.MustParse("018681fd-5770-336f-227c-259435D7fc6b"),
	Name:       "logistics-api",
	ExternalId: "dummy/logistics-api",
	Type:       "Application",
	Status:     types.ComponentStatusHealthy,
	Labels:     types.JSONStringMap{"telemetry": "enabled"},
	Owner:      "logistics-team",
	ParentId:   &Logistics.ID,
	Path:       Logistics.ID.String(),
	CreatedAt:  DummyCreatedAt,
}

var LogisticsUI = models.Component{
	ID:         uuid.MustParse("018681fd-c1ff-16ee-dff0-8c8796e4263e"),
	Name:       "logistics-ui",
	Type:       "Application",
	ExternalId: "dummy/logistics-ui",
	Status:     types.ComponentStatusHealthy,
	Owner:      "logistics-team",
	ParentId:   &Logistics.ID,
	Path:       Logistics.ID.String(),
	CreatedAt:  DummyCreatedAt,
}

var LogisticsWorker = models.Component{
	ID:         uuid.MustParse("018681fe-010a-6647-74ad-58b3a136dfe4"),
	Name:       "logistics-worker",
	ExternalId: "dummy/logistics-worker",
	Type:       "Application",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &LogisticsAPI.ID,
	Path:       Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
	CreatedAt:  DummyCreatedAt,
}

var LogisticsDB = models.Component{
	ID:           uuid.MustParse("018681fe-4529-c50f-26fd-530fa9c57319"),
	Name:         "logistics-db",
	ExternalId:   "dummy/logistics-db",
	Type:         "Database",
	Status:       types.ComponentStatusUnhealthy,
	StatusReason: "database not accepting connections",
	ParentId:     &LogisticsAPI.ID,
	Path:         Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
	CreatedAt:    DummyCreatedAt,
}

var ClusterComponent = models.Component{
	ID:         uuid.MustParse("018681fe-8156-4b91-d178-caf8b3c2818c"),
	Name:       "cluster",
	ExternalId: "dummy/cluster",
	Type:       "KubernetesCluster",
	Status:     types.ComponentStatusHealthy,
	CreatedAt:  DummyCreatedAt,
	Tooltip:    "Kubernetes Cluster",
	Icon:       "icon-cluster",
}

var NodesComponent = models.Component{
	ID:         uuid.MustParse("018681fe-b27e-7627-72c2-ad18e93f72f4"),
	Name:       "Nodes",
	Icon:       "icon-kubernetes-node",
	Tooltip:    "Kubernetes Nodes",
	ExternalId: "dummy/nodes",
	Type:       "KubernetesNodes",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       ClusterComponent.ID.String(),
}

var NodeA = models.Component{
	ID:         uuid.MustParse("018681fe-f5aa-37e9-83f7-47b5b0232d5e"),
	Name:       "node-a",
	Icon:       "icon-kubernetes-node",
	Tooltip:    "Node A",
	ExternalId: "dummy/node-a",
	Type:       "KubernetesNode",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       fmt.Sprintf("%s.%s", ClusterComponent.ID.String(), NodesComponent.ID.String()),
}

var NodeB = models.Component{
	ID:         uuid.MustParse("018681ff-227e-4d71-b38e-0693cc862213"),
	Name:       "node-b",
	Icon:       "icon-kubernetes-node",
	Tooltip:    "Node B",
	ExternalId: "dummy/node-b",
	Type:       "KubernetesNode",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       fmt.Sprintf("%s.%s", ClusterComponent.ID.String(), NodesComponent.ID.String()),
}

var PodsComponent = models.Component{
	ID:         uuid.MustParse("018681ff-559f-7183-19d1-7d898b4e1413"),
	Name:       "Pods",
	Icon:       "icon-kubernetes-pod",
	Tooltip:    "Kubernetes Pods",
	ExternalId: "dummy/pods",
	Type:       "KubernetesPods",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       ClusterComponent.ID.String(),
}

var LogisticsAPIPod = models.Component{
	ID:         uuid.MustParse("018681ff-80ed-d10d-21ef-c74f152b085b"),
	Name:       "logistics-api-7df4c7f6b7-x9k2m",
	Icon:       "icon-kubernetes-pod",
	Tooltip:    "Logistic API Pod",
	ExternalId: "dummy/logistics-api-7df4c7f6b7-x9k2m",
	Type:       "KubernetesPod",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       fmt.Sprintf("%s.%s", ClusterComponent.ID.String(), PodsComponent.ID.String()),
	Properties: []*models.Property{{Name: "memory", Unit: "bytes", Value: lo.ToPtr(int64(100))}},
}

var LogisticsUIPod = models.Component{
	ID:         uuid.MustParse("018681ff-b6c1-a14d-2fd4-8c7dac94cddd"),
	Name:       "logistics-ui-6c8f9b4d5e-m7n8p",
	Icon:       "icon-kubernetes-pod",
	Tooltip:    "Logistic UI Pod",
	Type:       "KubernetesPod",
	ExternalId: "dummy/logistics-ui-6c8f9b4d5e-m7n8p",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       fmt.Sprintf("%s.%s", ClusterComponent.ID.String(), PodsComponent.ID.String()),
	Properties: []*models.Property{{Name: "memory", Unit: "bytes", Value: lo.ToPtr(int64(200))}},
}

var LogisticsWorkerPod = models.Component{
	ID:         uuid.MustParse("018681ff-e578-a926-e366-d2dc0646eafa"),
	Name:       "logistics-worker-79cb67d8f5-lr66n",
	Icon:       "icon-kubernetes-pod",
	Tooltip:    "Logistic Worker Pod",
	ExternalId: "dummy/logistics-worker-79cb67d8f5-lr66n",
	Type:       "KubernetesPod",
	Status:     types.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  DummyCreatedAt,
	Path:       fmt.Sprintf("%s.%s", ClusterComponent.ID.String(), PodsComponent.ID.String()),
	Properties: []*models.Property{{Name: "memory", Unit: "bytes", Value: lo.ToPtr(int64(300))}},
}

var PaymentsAPI = models.Component{
	ID:         uuid.MustParse("4643e4de-6215-4c71-9600-9cf69b2cbbee"),
	AgentID:    GCPAgent.ID,
	Name:       "payments-api",
	ExternalId: "dummy/payments-api",
	Type:       "Application",
	CreatedAt:  DummyCreatedAt,
	Status:     types.ComponentStatusHealthy,
}

var FluxComponent = models.Component{
	ID:         uuid.MustParse("018cb576-11e3-a43a-75fd-3cbf5c8c804a"),
	Name:       "flux",
	ExternalId: "dummy/flux",
	Type:       "Flux",
	CreatedAt:  DummyCreatedAtPlus3Years,
	Labels:     types.JSONStringMap{"fluxcd.io/name": "flux"},
	Status:     types.ComponentStatusHealthy,
}

var KustomizeComponent = models.Component{
	ID:         uuid.MustParse("018cb576-4c81-91da-e59d-f25464b8bf91"),
	Name:       "kustomize-component",
	ExternalId: "dummy/kustomize-component",
	Type:       "FluxKustomize",
	CreatedAt:  DummyCreatedAt,
	ParentId:   &FluxComponent.ID,
	Status:     types.ComponentStatusHealthy,
	Properties: []*models.Property{{Name: "name", Text: "kustomize"}},
}

var KustomizeFluxComponent = models.Component{
	ID:         uuid.MustParse("018cb576-8036-10d8-edf1-cb49be2c0d93"),
	Name:       "kustomize-flux-component",
	ExternalId: "dummy/kustomize-flux-component",
	Type:       "Application",
	CreatedAt:  DummyCreatedAt,
	Status:     types.ComponentStatusHealthy,
	ParentId:   &KustomizeComponent.ID,
	Selectors: types.ResourceSelectors{
		{LabelSelector: "fluxcd.io/name=flux"},
	},
}

// Order is important since ParentIDs refer to previous components
var AllDummyComponents = []models.Component{
	Logistics,
	LogisticsAPI,
	LogisticsUI,
	LogisticsWorker,
	LogisticsDB,
	ClusterComponent,
	NodesComponent,
	PodsComponent,
	NodeA,
	NodeB,
	LogisticsAPIPod,
	LogisticsUIPod,
	LogisticsWorkerPod,
	PaymentsAPI,
	FluxComponent,
	KustomizeComponent,
	KustomizeFluxComponent,
}
