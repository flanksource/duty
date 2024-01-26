package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPIPodLogFile = models.Artifact{
	ID:       uuid.MustParse("018d411b-a35c-9d53-d223-454bdb173569"),
	Path:     "/logs/pods",
	Filename: "logistics-api.txt",
	Size:     1024,
}

var AllDummyArtifacts = []models.Artifact{
	LogisticsAPIPodLogFile,
}
