package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

var Logistics = models.Component{
	ID:         uuid.MustParse("018681fc-e54f-bd4f-42be-068a9a69eeb5"),
	Name:       "logistics",
	Type:       "Entity",
	ExternalId: "dummy/logistics",
	Labels:     types.JSONStringMap{"telemetry": "enabled"},
	Owner:      "logistics-team",
	CreatedAt:  models.LocalTime(DummyCreatedAt),
	Status:     models.ComponentStatusHealthy,
}

var LogisticsAPI = models.Component{
	ID:         uuid.MustParse("018681fd-5770-336f-227c-259435D7fc6b"),
	Name:       "logistics-api",
	ExternalId: "dummy/logistics-api",
	Type:       "Application",
	Status:     models.ComponentStatusHealthy,
	Labels:     types.JSONStringMap{"telemetry": "enabled"},
	Owner:      "logistics-team",
	ParentId:   &Logistics.ID,
	Path:       Logistics.ID.String(),
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsUI = models.Component{
	ID:         uuid.MustParse("018681fd-c1ff-16ee-dff0-8c8796e4263e"),
	Name:       "logistics-ui",
	Type:       "Application",
	ExternalId: "dummy/logistics-ui",
	Status:     models.ComponentStatusHealthy,
	Owner:      "logistics-team",
	ParentId:   &Logistics.ID,
	Path:       Logistics.ID.String(),
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsWorker = models.Component{
	ID:         uuid.MustParse("018681fe-010a-6647-74ad-58b3a136dfe4"),
	Name:       "logistics-worker",
	ExternalId: "dummy/logistics-worker",
	Type:       "Application",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &LogisticsAPI.ID,
	Path:       Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsDB = models.Component{
	ID:         uuid.MustParse("018681fe-4529-c50f-26fd-530fa9c57319"),
	Name:       "logistics-db",
	ExternalId: "dummy/logistics-db",
	Type:       "Database",
	Status:     models.ComponentStatusUnhealthy,
	ParentId:   &LogisticsAPI.ID,
	Path:       Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var ClusterComponent = models.Component{
	ID:         uuid.MustParse("018681fe-8156-4b91-d178-caf8b3c2818c"),
	Name:       "cluster",
	ExternalId: "dummy/cluster",
	Type:       "KubernetesCluster",
	Status:     models.ComponentStatusHealthy,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodesComponent = models.Component{
	ID:         uuid.MustParse("018681fe-b27e-7627-72c2-ad18e93f72f4"),
	Name:       "Nodes",
	ExternalId: "dummy/nodes",
	Type:       "KubernetesNodes",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodeA = models.Component{
	ID:         uuid.MustParse("018681fe-f5aa-37e9-83f7-47b5b0232d5e"),
	Name:       "node-a",
	ExternalId: "dummy/node-a",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodeB = models.Component{
	ID:         uuid.MustParse("018681ff-227e-4d71-b38e-0693cc862213"),
	Name:       "node-b",
	ExternalId: "dummy/node-b",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var PodsComponent = models.Component{
	ID:         uuid.MustParse("018681ff-559f-7183-19d1-7d898b4e1413"),
	Name:       "Pods",
	ExternalId: "dummy/pods",
	Type:       "KubernetesPods",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsAPIPod = models.Component{
	ID:         uuid.MustParse("018681ff-80ed-d10d-21ef-c74f152b085b"),
	Name:       "logistics-api-574dc95b5d-mp64w",
	ExternalId: "dummy/logistics-api-574dc95b5d-mp64w",
	Type:       "KubernetesPod",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsUIPod = models.Component{
	ID:         uuid.MustParse("018681ff-b6c1-a14d-2fd4-8c7dac94cddd"),
	Name:       "logistics-ui-676b85b87c-tjjcp",
	Type:       "KubernetesPod",
	ExternalId: "dummy/logistics-ui-676b85b87c-tjjcp",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsWorkerPod = models.Component{
	ID:         uuid.MustParse("018681ff-e578-a926-e366-d2dc0646eafa"),
	Name:       "logistics-worker-79cb67d8f5-lr66n",
	ExternalId: "dummy/logistics-worker-79cb67d8f5-lr66n",
	Type:       "KubernetesPod",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
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
}
