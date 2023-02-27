package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsDBErrorEvidence = models.Evidence{
	ID:           uuid.New(),
	HypothesisID: LogisticsAPIDownHypothesis.ID,
	ComponentID:  &LogisticsDB.ID,
	CreatedBy:    JohnDoe.ID,
	Description:  "Logisctics DB attached component",
	Type:         "component",
}

var AllDummyEvidences = []models.Evidence{LogisticsDBErrorEvidence}
