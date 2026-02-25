package connection

import (
	"fmt"

	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type Loki struct {
	ConnectionName string `json:"connection,omitempty"`

	URL      string        `json:"url,omitempty"`
	Username *types.EnvVar `json:"username,omitempty"`
	Password *types.EnvVar `json:"password,omitempty"`
}

func (c *Loki) Populate(ctx ConnectionContext) error {
	if c.ConnectionName != "" {
		conn, err := ctx.HydrateConnectionByURL(c.ConnectionName)
		if err != nil {
			return fmt.Errorf("could not hydrate connection[%s]: %w", c.ConnectionName, err)
		} else if conn == nil {
			return fmt.Errorf("connection[%s] not found", c.ConnectionName)
		}

		if c.URL == "" && conn.URL != "" {
			c.URL = conn.URL
		}

		if c.Username == nil || c.Username.IsEmpty() {
			c.Username = &types.EnvVar{ValueStatic: conn.Username}
		}
		if c.Password == nil || c.Password.IsEmpty() {
			c.Password = &types.EnvVar{ValueStatic: conn.Password}
		}
	}

	if c.Username != nil && !c.Username.IsEmpty() {
		if v, err := ctx.GetEnvValueFromCache(*c.Username, ctx.GetNamespace()); err != nil {
			return fmt.Errorf("could not get username from env var: %w", err)
		} else {
			c.Username.ValueStatic = v
		}
	}

	if c.Password != nil && !c.Password.IsEmpty() {
		if v, err := ctx.GetEnvValueFromCache(*c.Password, ctx.GetNamespace()); err != nil {
			return fmt.Errorf("could not get password from env var: %w", err)
		} else {
			c.Password.ValueStatic = v
		}
	}

	return nil
}
