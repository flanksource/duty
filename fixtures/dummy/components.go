package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var Logistics = models.Component{
	ID:         uuid.New(),
	Name:       "logistics",
	Type:       "Entity",
	ExternalId: "dummy/logistics",
	Status:     models.ComponentStatusHealthy,
}

var LogisticsAPI = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-api",
	ExternalId: "dummy/logistics-api",
	Type:       "Application",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &Logistics.ID,
}

var LogisticsUI = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-ui",
	Type:       "Application",
	ExternalId: "dummy/logistics-ui",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &Logistics.ID,
}

var LogisticsWorker = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-worker",
	ExternalId: "dummy/logistics-worker",
	Type:       "Application",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &LogisticsAPI.ID,
}

var LogisticsDB = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-db",
	ExternalId: "dummy/logistics-db",
	Type:       "Database",
	Status:     models.ComponentStatusUnhealthy,
	ParentId:   &LogisticsAPI.ID,
}

var ClusterComponent = models.Component{
	ID:         uuid.New(),
	Name:       "cluster",
	ExternalId: "dummy/cluster",
	Type:       "KubernetesCluster",
	Status:     models.ComponentStatusHealthy,
}

var NodeA = models.Component{
	ID:         uuid.New(),
	Name:       "node-a",
	ExternalId: "dummy/node-a",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
}

var NodeB = models.Component{
	ID:         uuid.New(),
	Name:       "node-b",
	ExternalId: "dummy/node-b",
	Type:       "KubernetesNode",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &ClusterComponent.ID,
}

var LogisticsAPIPod = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-api-574dc95b5d-mp64w",
	ExternalId: "dummy/logistics-api-574dc95b5d-mp64w",
	Type:       "KubernetesPod",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodeA.ID,
}

var LogisticsUIPod = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-ui-676b85b87c-tjjcp",
	Type:       "KubernetesPod",
	ExternalId: "dummy/logistics-ui-676b85b87c-tjjcp",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodeA.ID,
}

var LogisticsWorkerPod = models.Component{
	ID:         uuid.New(),
	Name:       "logistics-worker-79cb67d8f5-lr66n",
	ExternalId: "dummy/logistics-worker-79cb67d8f5-lr66n",
	Type:       "KubernetesPod",
	Status:     models.ComponentStatusHealthy,
	ParentId:   &NodeB.ID,
}

// Order is important since ParentIDs refer to previous components
var AllDummyComponents = []models.Component{
	Logistics,
	LogisticsAPI,
	LogisticsUI,
	LogisticsWorker,
	LogisticsDB,
	ClusterComponent,
	NodeA,
	NodeB,
	LogisticsAPIPod,
	LogisticsUIPod,
	LogisticsWorkerPod,
}
