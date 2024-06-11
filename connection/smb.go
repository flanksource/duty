package connection

import (
	"strconv"

	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type SMBConnection struct {
	// ConnectionName of the connection. It'll be used to populate the connection fields.
	ConnectionName string `yaml:"connection,omitempty" json:"connection,omitempty"`
	//Port on which smb server is running. Defaults to 445
	Port                 int `yaml:"port,omitempty" json:"port,omitempty"`
	types.Authentication `yaml:",inline" json:",inline"`
	//Domain...
	Domain string `yaml:"domain,omitempty" json:"domain,omitempty"`
}

func (c SMBConnection) GetPort() int {
	if c.Port != 0 {
		return c.Port
	}
	return 445
}

func (c *SMBConnection) HydrateConnection(ctx ConnectionContext) (found bool, err error) {
	connection, err := ctx.HydrateConnectionByURL(c.ConnectionName)
	if err != nil {
		return false, err
	}

	if connection == nil {
		return false, nil
	}

	c.Authentication = types.Authentication{
		Username: types.EnvVar{ValueStatic: connection.Username},
		Password: types.EnvVar{ValueStatic: connection.Password},
	}

	if domain, ok := connection.Properties["domain"]; ok {
		c.Domain = domain
	}

	if portRaw, ok := connection.Properties["port"]; ok {
		if port, err := strconv.Atoi(portRaw); nil == err {
			c.Port = port
		}
	}

	return true, nil
}
