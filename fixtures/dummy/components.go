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
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsUI = models.Component{
	ID:         uuid.MustParse("018681FD-C1FF-16EE-DFF0-8C8796E4263E"),
	Name:       "logistics-ui",
	Type:       "Application",
	ExternalId: "dummy/logistics-ui",
	Status:     models.ComponentStatusHealthy,
	Owner:      "logistics-team",
	ParentId:   &Logistics.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsWorker = models.Component{
	ID:         uuid.MustParse("018681FE-010A-6647-74AD-58B3A136DFE4"),
	Name:       "logistics-worker",
	ExternalId: "dummy/logistics-worker",
	Type:       "Application",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &LogisticsAPI.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsDB = models.Component{
	ID:         uuid.MustParse("018681FE-4529-C50F-26FD-530FA9C57319"),
	Name:       "logistics-db",
	ExternalId: "dummy/logistics-db",
	Type:       "Database",
	Status:     models.ComponentStatusUnhealthy,
	ParentId:   &LogisticsAPI.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var ClusterComponent = models.Component{
	ID:         uuid.MustParse("018681FE-8156-4B91-D178-CAF8B3C2818C"),
	Name:       "cluster",
	ExternalId: "dummy/cluster",
	Type:       "KubernetesCluster",
	Status:     models.ComponentStatusHealthy,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodesComponent = models.Component{
	ID:         uuid.MustParse("018681FE-B27E-7627-72C2-AD18E93F72F4"),
	Name:       "Nodes",
	ExternalId: "dummy/nodes",
	Type:       "KubernetesNodes",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodeA = models.Component{
	ID:         uuid.MustParse("018681FE-F5AA-37E9-83F7-47B5B0232D5E"),
	Name:       "node-a",
	ExternalId: "dummy/node-a",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var NodeB = models.Component{
	ID:         uuid.MustParse("018681FF-227E-4D71-B38E-0693CC862213"),
	Name:       "node-b",
	ExternalId: "dummy/node-b",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodesComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var PodsComponent = models.Component{
	ID:         uuid.MustParse("018681FF-559F-7183-19D1-7D898B4E1413"),
	Name:       "Pods",
	ExternalId: "dummy/pods",
	Type:       "KubernetesPods",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsAPIPod = models.Component{
	ID:         uuid.MustParse("018681FF-80ED-D10D-21EF-C74F152B085B"),
	Name:       "logistics-api-574dc95b5d-mp64w",
	ExternalId: "dummy/logistics-api-574dc95b5d-mp64w",
	Type:       "KubernetesPod",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsUIPod = models.Component{
	ID:         uuid.MustParse("018681FF-B6C1-A14D-2FD4-8C7DAC94CDDD"),
	Name:       "logistics-ui-676b85b87c-tjjcp",
	Type:       "KubernetesPod",
	ExternalId: "dummy/logistics-ui-676b85b87c-tjjcp",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &PodsComponent.ID,
	CreatedAt:  models.LocalTime(DummyCreatedAt),
}

var LogisticsWorkerPod = models.Component{
	ID:         uuid.MustParse("018681FF-E578-A926-E366-D2DC0646EAFA"),
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
