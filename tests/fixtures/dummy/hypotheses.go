package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPIDownHypothesis = models.Hypothesis{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Title:      "Logistics DB database error hypothesis",
	CreatedBy:  JohnDoe.ID,
	Type:       "solution",
	Status:     "possible",
}

var AllDummyHypotheses = []models.Hypothesis{LogisticsAPIDownHypothesis}
