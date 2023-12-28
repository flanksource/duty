package dummy

import (
	"github.com/flanksource/duty/models"
)

var LogisticsAPIPodNodeAComponentRelationship = models.ComponentRelationship{
	ComponentID:    LogisticsAPIPod.ID,
	RelationshipID: NodeA.ID,
}

var LogisticsUIPodNodeAComponentRelationship = models.ComponentRelationship{
	ComponentID:    LogisticsUIPod.ID,
	RelationshipID: NodeA.ID,
}

var LogisticsWorkerPodNodeBComponentRelationship = models.ComponentRelationship{
	ComponentID:    LogisticsWorkerPod.ID,
	RelationshipID: NodeB.ID,
}

// Create a cyclic loop
var FluxKustomizeFluxComponentRelationship = models.ComponentRelationship{
	ComponentID:    FluxComponent.ID,
	RelationshipID: KustomizeFluxComponent.ID,
}

var AllDummyComponentRelationships = []models.ComponentRelationship{
	LogisticsAPIPodNodeAComponentRelationship,
	LogisticsUIPodNodeAComponentRelationship,
	LogisticsWorkerPodNodeBComponentRelationship,
	FluxKustomizeFluxComponentRelationship,
}
