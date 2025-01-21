package secret

import (
	"fmt"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
	"gocloud.dev/secrets"
)

var allowedConnectionTypes = []string{
	models.ConnectionTypeAWSKMS,
	models.ConnectionTypeGCPKMS,
	models.ConnectionTypeAzureKeyVault,
	// Vault not supported yet
}

func KeeperFromConnection(ctx context.Context, connectionString string) (*secrets.Keeper, error) {
	conn, err := ctx.HydrateConnectionByURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to hydrate connection: %w", err)
	} else if conn == nil {
		return nil, fmt.Errorf("connection not found: %s", connectionString)
	}

	if !lo.Contains(allowedConnectionTypes, conn.Type) {
		return nil, fmt.Errorf("connection type %s cannot be used to create a SecretKeeper", conn.Type)
	}

	switch conn.Type {
	case models.ConnectionTypeAWSKMS:
		var kmsConn connection.AWSKMS
		kmsConn.FromModel(*conn)
		return kmsConn.SecretKeeper(ctx)

	case models.ConnectionTypeAzureKeyVault:
		var keyvaultConn connection.AzureKeyVault
		keyvaultConn.FromModel(*conn)
		return keyvaultConn.SecretKeeper(ctx)

	case models.ConnectionTypeGCPKMS:
		var kmsConn connection.GCPKMS
		kmsConn.FromModel(*conn)
		return kmsConn.SecretKeeper(ctx)
	}

	return nil, nil
}
