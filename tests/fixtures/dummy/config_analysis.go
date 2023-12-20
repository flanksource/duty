package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

func AllDummyConfigAnalysis() []models.ConfigAnalysis {
	return []models.ConfigAnalysis{
		{
			ID:            uuid.New(),
			ConfigID:      LogisticsDBRDS.ID,
			AnalysisType:  models.AnalysisTypeSecurity,
			Severity:      "critical",
			Message:       "Port exposed to public",
			FirstObserved: &CurrentTime,
			Status:        models.AnalysisStatusOpen,
		},
		{
			ID:            uuid.New(),
			ConfigID:      EC2InstanceB.ID,
			AnalysisType:  models.AnalysisTypeSecurity,
			Severity:      "critical",
			Message:       "SSH key not rotated",
			FirstObserved: &CurrentTime,
			Status:        models.AnalysisStatusOpen,
		},
	}
}
