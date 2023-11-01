package dummy

import (
	"github.com/flanksource/duty/models"
)

var LogisticsDBCheckComponentRelationship = models.CheckComponentRelationship{
	ComponentID: LogisticsDB.ID,
	CheckID:     LogisticsDBCheck.ID,
	CanaryID:    LogisticsDBCheck.CanaryID,
}

var LogisticsAPIHealthHTTPCheckComponentRelationship = models.CheckComponentRelationship{
	ComponentID: LogisticsAPI.ID,
	CheckID:     LogisticsAPIHealthHTTPCheck.ID,
	CanaryID:    LogisticsAPIHealthHTTPCheck.CanaryID,
}

var LogisticsAPIHomeHTTPCheckComponentRelationship = models.CheckComponentRelationship{
	ComponentID: LogisticsAPI.ID,
	CheckID:     LogisticsAPIHomeHTTPCheck.ID,
	CanaryID:    LogisticsAPIHomeHTTPCheck.CanaryID,
}

var AllDummyCheckComponentRelationships = []models.CheckComponentRelationship{
	LogisticsDBCheckComponentRelationship,
	LogisticsAPIHealthHTTPCheckComponentRelationship,
	LogisticsAPIHomeHTTPCheckComponentRelationship,
}
