package connection

import (
	"fmt"

	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type AWSConnection struct {
	// ConnectionName of the connection. It'll be used to populate the endpoint, accessKey and secretKey.
	ConnectionName string       `yaml:"connection,omitempty" json:"connection,omitempty"`
	AccessKey      types.EnvVar `yaml:"accessKey" json:"accessKey,omitempty"`
	SecretKey      types.EnvVar `yaml:"secretKey" json:"secretKey,omitempty"`
	SessionToken   types.EnvVar `yaml:"sessionToken,omitempty" json:"sessionToken,omitempty"`
	Region         string       `yaml:"region,omitempty" json:"region,omitempty"`
	Endpoint       string       `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	// Skip TLS verify when connecting to aws
	SkipTLSVerify bool `yaml:"skipTLSVerify,omitempty" json:"skipTLSVerify,omitempty"`
}

func (t *AWSConnection) GetUsername() types.EnvVar {
	return t.AccessKey
}

func (t *AWSConnection) GetPassword() types.EnvVar {
	return t.SecretKey
}

func (t *AWSConnection) GetProperties() map[string]string {
	return map[string]string{
		"region": t.Region,
	}
}

func (t *AWSConnection) GetURL() types.EnvVar {
	return types.EnvVar{ValueStatic: t.Endpoint}
}

// Populate populates an AWSConnection with credentials and other information.
// If a connection name is specified, it'll be used to populate the endpoint, accessKey and secretKey.
func (t *AWSConnection) Populate(ctx ConnectionContext) error {
	if t.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(t.ConnectionName)
		if err != nil {
			return fmt.Errorf("could not parse EC2 access key: %v", err)
		}

		t.AccessKey.ValueStatic = connection.Username
		t.SecretKey.ValueStatic = connection.Password
		if t.Endpoint == "" {
			t.Endpoint = connection.URL
		}

		t.SkipTLSVerify = connection.InsecureTLS
		if t.Region == "" {
			if region, ok := connection.Properties["region"]; ok {
				t.Region = region
			}
		}
	}

	if accessKey, err := ctx.GetEnvValueFromCache(t.AccessKey, ""); err != nil {
		return fmt.Errorf("could not parse AWS access key id: %v", err)
	} else {
		t.AccessKey.ValueStatic = accessKey
	}

	if secretKey, err := ctx.GetEnvValueFromCache(t.SecretKey, ""); err != nil {
		return fmt.Errorf(fmt.Sprintf("Could not parse AWS secret access key: %v", err))
	} else {
		t.SecretKey.ValueStatic = secretKey
	}

	if sessionToken, err := ctx.GetEnvValueFromCache(t.SessionToken, ""); err != nil {
		return fmt.Errorf(fmt.Sprintf("Could not parse AWS session token: %v", err))
	} else {
		t.SessionToken.ValueStatic = sessionToken
	}

	return nil
}
