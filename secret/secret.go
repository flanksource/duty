package secret

import (
	gocontext "context"
	"encoding/base64"
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

func DecryptB64WithConnection(ctx context.Context, secretKeeperConnection string, b64Ciphertext string) (Sensitive, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(b64Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 ciphertext: %w", err)
	}

	return DecryptWithConnection(ctx, secretKeeperConnection, ciphertext)
}

func DecryptWithConnection(ctx context.Context, secretKeeperConnection string, ciphertext []byte) (Sensitive, error) {
	keeper, err := KeeperFromConnection(ctx, secretKeeperConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret keeper from connection (%s): %w", secretKeeperConnection, err)
	}
	defer keeper.Close()

	return Decrypt(ctx, keeper, ciphertext)
}

func Decrypt(ctx gocontext.Context, keeper *secrets.Keeper, ciphertext []byte) (Sensitive, error) {
	decrypted, err := keeper.Decrypt(ctx, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return Sensitive(decrypted), nil
}
