package connection

import (
	"fmt"
	"strconv"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type SFTPConnection struct {
	// ConnectionName of the connection. It'll be used to populate the connection fields.
	ConnectionName string `yaml:"connection,omitempty" json:"connection,omitempty"`
	// Port for the SSH server. Defaults to 22
	Port                 int    `yaml:"port,omitempty" json:"port,omitempty"`
	Host                 string `yaml:"host,omitempty" json:"host,omitempty"`
	types.Authentication `yaml:",inline" json:",inline"`
}

func (c SFTPConnection) ToModel() models.Connection {
	return models.Connection{
		Type:     models.ConnectionTypeSFTP,
		Username: c.GetUsername(),
		Password: c.GetPassword(),
		URL:      c.Host,
		Properties: types.JSONStringMap{
			"port": fmt.Sprintf("%d", c.GetPort()),
		},
	}
}

func (c *SFTPConnection) HydrateConnection(ctx ConnectionContext) (err error) {
	if c.ConnectionName != "" {
		conn, err := ctx.HydrateConnectionByURL(c.ConnectionName)
		if err != nil {
			return err
		}

		if c.Username.IsEmpty() {
			c.Username = types.EnvVar{ValueStatic: conn.Username}
		}
		if c.Password.IsEmpty() {
			c.Password = types.EnvVar{ValueStatic: conn.Password}
		}

		if c.Port == 0 {
			if port, ok := conn.Properties["port"]; ok {
				if p, err := strconv.Atoi(port); err == nil {
					c.Port = p
				}
			}
		}

		if c.Port == 0 {
			c.Port = 22
		}

		if c.Host == "" && conn.URL != "" {
			c.Host = conn.URL
		}
	}

	if username, err := ctx.GetEnvValueFromCache(c.Username, ctx.GetNamespace()); err != nil {
		return fmt.Errorf("could not parse username: %v", err)
	} else {
		c.Username.ValueStatic = username
	}

	if password, err := ctx.GetEnvValueFromCache(c.Password, ctx.GetNamespace()); err != nil {
		return fmt.Errorf("could not parse password: %w", err)
	} else {
		c.Password.ValueStatic = password
	}

	return nil
}

func (c SFTPConnection) GetPort() int {
	if c.Port != 0 {
		return c.Port
	}
	return 22
}
