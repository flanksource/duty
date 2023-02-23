package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var currentTime = time.Now()

var LogisticsDBRDSAnalysis = models.ConfigAnalysis{
	ID:            uuid.New(),
	ConfigID:      LogisticsDBRDS.ID,
	AnalysisType:  "security",
	Severity:      "critical",
	Message:       "Port exposed to public",
	FirstObserved: &currentTime,
}

var EC2InstanceBAnalysis = models.ConfigAnalysis{
	ID:            uuid.New(),
	ConfigID:      EC2InstanceB.ID,
	AnalysisType:  "security",
	Severity:      "critical",
	Message:       "SSH key not rotated",
	FirstObserved: &currentTime,
}

var AllDummyConfigAnalysis = []models.ConfigAnalysis{
	LogisticsDBRDSAnalysis,
	EC2InstanceBAnalysis,
}
