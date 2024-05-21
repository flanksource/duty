package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var JobHistoryCanarySuccess = models.JobHistory{
	ID:           uuid.MustParse("018f9b11-7639-c442-4b62-92fe7fdd4f5c"),
	Name:         "Canary1",
	ResourceType: "canary",
	Status:       models.StatusSuccess,
}

var JobHistoryCanaryFailure = models.JobHistory{
	ID:           uuid.MustParse("018f9b10-ecb8-75c1-fefa-9d589fb99c65"),
	Name:         "Canary1",
	ResourceType: "canary",
	Status:       models.StatusFailed,
}

var JobHistoryCanaryWarning = models.JobHistory{
	ID:           uuid.MustParse("018f9b11-188c-4d71-c826-e356b535ed96"),
	Name:         "Canary1",
	ResourceType: "canary",
	Status:       models.StatusWarning,
}

var JobHistoryTopologySuccess = models.JobHistory{
	ID:           uuid.MustParse("018f9b11-40f7-e43e-cef5-f9441d77ce5b"),
	Name:         "Topology1",
	ResourceType: "topology",
	Status:       models.StatusSuccess,
}

var AllDummyJobHistories = []models.JobHistory{
	JobHistoryCanarySuccess,
	JobHistoryCanaryFailure,
	JobHistoryCanaryWarning,
	JobHistoryTopologySuccess,
}
