package connection

import (
	"context"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

type ConnectionContext interface {
	context.Context
	HydrateConnectionByURL(connectionName string) (*models.Connection, error)
	GetEnvValueFromCache(env types.EnvVar, namespace string) (string, error)
	GetNamespace() string
}
