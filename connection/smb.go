package connection

import (
	"fmt"
	"strconv"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type SMBConnection struct {
	types.Authentication `yaml:",inline" json:",inline"`
	// ConnectionName of the connection. It'll be used to populate the connection fields.
	ConnectionName string `yaml:"connection,omitempty" json:"connection,omitempty"`
	// Port on which smb server is running. Defaults to 445
	Port   int    `yaml:"port,omitempty" json:"port,omitempty"`
	Domain string `yaml:"domain,omitempty" json:"domain,omitempty"`
	Share  string `yaml:"share,omitempty" json:"share,omitempty"`
}

func (c SMBConnection) GetPort() int {
	if c.Port != 0 {
		return c.Port
	}

	return 445
}

func (c SMBConnection) ToModel() models.Connection {
	return models.Connection{
		Type:     models.ConnectionTypeSMB,
		Username: c.GetUsername(),
		Password: c.GetPassword(),
		URL:      c.Domain,
		Properties: types.JSONStringMap{
			"port":  fmt.Sprintf("%d", c.GetPort()),
			"share": c.Share,
		},
	}
}

func (c *SMBConnection) Populate(ctx ConnectionContext) error {
	if c.ConnectionName != "" {
		conn, err := ctx.HydrateConnectionByURL(c.ConnectionName)
		if err != nil {
			return err
		}

		c.Username = types.EnvVar{ValueStatic: conn.Username}
		c.Password = types.EnvVar{ValueStatic: conn.Password}

		if c.Port == 0 {
			if port, ok := conn.Properties["port"]; ok {
				if p, err := strconv.Atoi(port); err == nil {
					c.Port = p
				}
			}
		}

		if domain, ok := conn.Properties["domain"]; ok {
			c.Domain = domain
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
