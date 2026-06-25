package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsDBRDSAnalysis = models.ConfigAnalysis{
	ID:            uuid.New(),
	ConfigID:      LogisticsDBRDS.ID,
	Analyzer:      "rds-port-exposed",
	AnalysisType:  models.AnalysisTypeSecurity,
	Severity:      models.SeverityCritical,
	Message:       "Port exposed to public",
	FirstObserved: &CurrentTime,
	Status:        models.AnalysisStatusOpen,
}

var EC2InstanceBAnalysis = models.ConfigAnalysis{
	ID:            uuid.New(),
	ConfigID:      EC2InstanceB.ID,
	Analyzer:      "ec2-ssh-key-not-rotated",
	AnalysisType:  models.AnalysisTypeSecurity,
	Severity:      models.SeverityCritical,
	Message:       "SSH key not rotated",
	FirstObserved: &CurrentTime,
	Status:        models.AnalysisStatusOpen,
}

func AllDummyConfigAnalysis() []models.ConfigAnalysis {
	return []models.ConfigAnalysis{
		LogisticsDBRDSAnalysis,
		EC2InstanceBAnalysis,
	}
}
