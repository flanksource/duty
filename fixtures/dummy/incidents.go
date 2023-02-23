package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPIDownIncident = models.Incident{
	ID:        uuid.New(),
	Title:     "Logistics API is down",
	CreatedBy: JohnDoe.ID,
	Type:      models.IncidentTypeAvailability,
	Status:    models.IncidentStatusOpen,
	Severity:  "Blocker",
}

var AllDummyIncidents = []models.Incident{LogisticsAPIDownIncident}
