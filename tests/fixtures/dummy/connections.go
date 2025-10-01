package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
)

var AWSConnection = models.Connection{
	ID:        uuid.MustParse("6c4c4c4c-0000-0000-0000-000000000001"),
	Name:      "aws-connection",
	Namespace: "default",
	Source:    models.SourceConfigFile,
	Type:      "aws",
}

var PostgresConnection = models.Connection{
	ID:        uuid.MustParse("6c4c4c4c-0000-0000-0000-000000000002"),
	Name:      "postgres-connection",
	Namespace: "production",
	Source:    models.SourceConfigFile,
	Type:      "postgres",
}

var AllDummyConnections = []models.Connection{AWSConnection, PostgresConnection}
