package context

import (
	"fmt"
	"time"

	dutyKubernetes "github.com/flanksource/duty/kubernetes"
)

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
}

func (c *KubernetesClient) SetExpiry(d time.Duration) {
	c.expiry = time.Now().Add(d)
}

func (c *KubernetesClient) RefreshWithExpiry(ctx Context, d time.Duration) error {
	if !c.HasExpired() {
		return nil
	}
	_, rc, err := c.Connection.Populate(ctx, true)
	if err != nil {
		return fmt.Errorf("error refreshing kubernetes client: %w", err)
	}

	// Update rest config in place for easy reuse
	c.Config.Host = rc.Host
	c.Config.TLSClientConfig = rc.TLSClientConfig
	c.Config.BearerToken = rc.BearerToken
	c.Config.BearerTokenFile = rc.BearerTokenFile
	c.Config.Username = rc.Username
	c.Config.Password = rc.Password

	c.SetExpiry(15 * time.Minute)
	return nil
}

func (c KubernetesClient) HasExpired() bool {
	if c.Connection.CanExpire() {
		return time.Until(c.expiry) <= 0
	}
	return false
}
