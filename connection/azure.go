package connection

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

	// BearerToken is populated from the referenced connection's
	// Properties["bearer"] when the connection has no Username/Password set.
	//
	// NOTE: Exported to avoid being flagged by CRD unexported-field validation.
	BearerToken string `json:"-" yaml:"-"`
}

// HydrateConnection attempts to find the connection by name
// and populate the endpoint and credentials.
func (g *AzureConnection) HydrateConnection(ctx ConnectionContext) error {
	connection, err := ctx.HydrateConnectionByURL(g.ConnectionName)
	if err != nil {
		return err
	}

	if connection != nil {
		if g.ClientID == nil || g.ClientID.IsEmpty() {
			g.ClientID = &types.EnvVar{ValueStatic: connection.Username}
		}
		if g.ClientSecret == nil || g.ClientSecret.IsEmpty() {
			g.ClientSecret = &types.EnvVar{ValueStatic: connection.Password}
		}
		if g.TenantID == "" {
			g.TenantID = connection.Properties["tenant"]
		}

		if connection.Username == "" && connection.Password == "" {
			g.BearerToken = connection.Properties["bearer"]
		}
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
	if connection.Username == "" && connection.Password == "" {
		g.BearerToken = connection.Properties["bearer"]
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
	if (g.ClientID == nil || g.ClientID.IsEmpty()) &&
		(g.ClientSecret == nil || g.ClientSecret.IsEmpty()) &&
		g.BearerToken != "" {
		return staticTokenCredential{token: g.BearerToken}, nil
	}
	return azidentity.NewClientSecretCredential(g.TenantID, g.ClientID.String(), g.ClientSecret.String(), nil)
}

type staticTokenCredential struct{ token string }

func (s staticTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: s.token, ExpiresOn: time.Now().Add(time.Hour)}, nil
}
