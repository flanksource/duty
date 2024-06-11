package connection

import (
	"strconv"

	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type SFTPConnection struct {
	// ConnectionName of the connection. It'll be used to populate the connection fields.
	ConnectionName string `yaml:"connection,omitempty" json:"connection,omitempty"`
	// Port for the SSH server. Defaults to 22
	Port                 int    `yaml:"port,omitempty" json:"port,omitempty"`
	Host                 string `yaml:"host" json:"host"`
	types.Authentication `yaml:",inline" json:",inline"`
}

func (c *SFTPConnection) HydrateConnection(ctx ConnectionContext) (found bool, err error) {
	connection, err := ctx.HydrateConnectionByURL(c.ConnectionName)
	if err != nil {
		return false, err
	}

	if connection == nil {
		return false, nil
	}

	c.Host = connection.URL
	c.Authentication = types.Authentication{
		Username: types.EnvVar{ValueStatic: connection.Username},
		Password: types.EnvVar{ValueStatic: connection.Password},
	}

	if portRaw, ok := connection.Properties["port"]; ok {
		if port, err := strconv.Atoi(portRaw); nil == err {
			c.Port = port
		}
	}

	return true, nil
}

func (c SFTPConnection) GetPort() int {
	if c.Port != 0 {
		return c.Port
	}
	return 22
}
