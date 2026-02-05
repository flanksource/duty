package connection

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gocloud.dev/secrets"
	"gocloud.dev/secrets/azurekeyvault"
)

type AzureKeyVault struct {
	AzureConnection `json:",inline"`

	// keyID is a URL to the key in the format
	// 	https://<vault-name>.vault.azure.net/keys/<key-name>
	KeyID string `json:"keyID,omitempty"`
}

func (t *AzureKeyVault) Populate(ctx ConnectionContext) error {
	return t.AzureConnection.HydrateConnection(ctx)
}

func (t *AzureKeyVault) FromModel(conn models.Connection) {
	t.AzureConnection.FromModel(conn)
	t.KeyID = conn.Properties["keyID"]
}

func (t *AzureKeyVault) SecretKeeper(ctx context.Context) (*secrets.Keeper, error) {
	creds, err := t.AzureConnection.TokenCredential()
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure token credential: %w", err)
	}

	clientMaker := func(keyVaultURI string) (*azkeys.Client, error) {
		return azkeys.NewClient(keyVaultURI, creds, &azkeys.ClientOptions{
			ClientOptions: policy.ClientOptions{},
		})
	}

	keeper, err := azurekeyvault.OpenKeeper(clientMaker, t.KeyID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Key Vault keeper: %w", err)
	}

	return keeper, nil
}
