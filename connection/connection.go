package connection

import (
	gocontext "context"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

type ConnectionContext interface {
	gocontext.Context
	HydrateConnectionByURL(connectionName string) (*models.Connection, error)
	GetEnvValueFromCache(env types.EnvVar, namespace string) (string, error)
	GetNamespace() string
}

func Get(ctx context.Context, connectionName string) (*models.Connection, error) {
	connection, err := context.FindConnectionByURL(ctx, connectionName)
	if err != nil {
		return nil, err
	} else if connection == nil {
		return nil, ctx.Oops().Code(api.ENOTFOUND).Errorf("connection (%s) not found", connectionName)
	}

	attr := models.ABACAttribute{Connection: *connection}
	if !rbac.HasPermission(ctx, ctx.Subject(), &attr, policy.ActionRead) {
		return nil, ctx.Oops().Code(api.EUNAUTHORIZED).
			Errorf("access denied to %s, `read` permission required on %s", ctx.Subject(), connectionName)
	}

	return context.HydrateConnection(ctx, connection)
}
