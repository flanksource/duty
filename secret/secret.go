package secret

import (
	"fmt"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"gocloud.dev/secrets"
)

var (
	// KMSConnection is the connection to the key management service
	// that's used to encrypt and decrypt secrets.
	KMSConnection string

	allowedConnectionTypes = []string{
		models.ConnectionTypeAWSKMS,
		models.ConnectionTypeGCPKMS,
		models.ConnectionTypeAzureKeyVault,
		// Vault not supported yet
	}
)

// TODO: Cache secret keepeer with TTL.

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

func Encrypt(ctx context.Context, sensitive Sensitive) (Ciphertext, error) {
	if KMSConnection == "" {
		return nil, oops.Errorf("secret keeper connection is not set")
	}

	keeper, err := KeeperFromConnection(ctx, KMSConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret keeper from connection (%s): %w", KMSConnection, err)
	}
	defer keeper.Close()

	ciphertext, err := keeper.Encrypt(ctx, []byte(sensitive.PlainText()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	return ciphertext, nil
}

func Decrypt(ctx context.Context, ciphertext Ciphertext) (Sensitive, error) {
	if KMSConnection == "" {
		return nil, oops.Errorf("secret keeper connection is not set")
	}

	keeper, err := KeeperFromConnection(ctx, KMSConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret keeper from connection (%s): %w", KMSConnection, err)
	}
	defer keeper.Close()

	decrypted, err := keeper.Decrypt(ctx, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return Sensitive(decrypted), nil
}
