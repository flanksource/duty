package connection

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type AzureConnection struct {
	ConnectionName string        `yaml:"connection,omitempty" json:"connection,omitempty"`
	ClientID       *types.EnvVar `yaml:"clientID,omitempty" json:"clientID,omitempty"`
	ClientSecret   *types.EnvVar `yaml:"clientSecret,omitempty" json:"clientSecret,omitempty"`
	TenantID       string        `yaml:"tenantID,omitempty" json:"tenantID,omitempty"`
}

// HydrateConnection attempts to find the connection by name
// and populate the endpoint and credentials.
func (g *AzureConnection) HydrateConnection(ctx ConnectionContext) error {
	connection, err := ctx.HydrateConnectionByURL(g.ConnectionName)
	if err != nil {
		return err
	}

	if connection != nil {
		g.ClientID = &types.EnvVar{ValueStatic: connection.Username}
		g.ClientSecret = &types.EnvVar{ValueStatic: connection.Password}
		g.TenantID = connection.Properties["tenant"]
	}

	return nil
}

func (g *AzureConnection) FromModel(connection models.Connection) {
	g.ConnectionName = connection.Name
	g.ClientID = &types.EnvVar{ValueStatic: connection.Username}
	g.ClientSecret = &types.EnvVar{ValueStatic: connection.Password}
	if tenantID, ok := connection.Properties["tenant"]; ok {
		g.TenantID = tenantID
	}
}

func (g AzureConnection) ToModel() models.Connection {
	return models.Connection{
		Type:     models.ConnectionTypeAzure,
		Name:     g.ConnectionName,
		Username: g.ClientID.String(),
		Password: g.ClientSecret.String(),
		Properties: types.JSONStringMap{
			"tenant": g.TenantID,
		},
	}
}

func (g *AzureConnection) TokenCredential() (azcore.TokenCredential, error) {
	return azidentity.NewClientSecretCredential(g.TenantID, g.ClientID.String(), g.ClientSecret.String(), nil)
}
