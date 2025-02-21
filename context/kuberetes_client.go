package context

import (
	"fmt"
	"time"

	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
}

var defaultExpiry = 15 * time.Minute

func NewKubernetesClient(ctx Context, conn KubernetesConnection) (*KubernetesClient, error) {
	c, rc, err := conn.Populate(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("error refreshing kubernetes client: %w", err)
	}
	client := &KubernetesClient{
		Client:     dutyKubernetes.NewKubeClient(c, rc),
		Connection: conn,
	}

	client.SetExpiry(defaultExpiry)
	return client, nil
}

func (c *KubernetesClient) SetExpiry(def time.Duration) {
	// Try parsing BearerToken as JWT and extract expiry
	if expiry := extractExpiryFromJWT(lo.FromPtr(c.Config).BearerToken); !expiry.IsZero() {
		c.expiry = expiry
	} else {
		c.expiry = time.Now().Add(def)
	}
}

func (c *KubernetesClient) Refresh(ctx Context) error {
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

	c.SetExpiry(defaultExpiry)
	return nil
}

func (c KubernetesClient) HasExpired() bool {
	if c.Connection.CanExpire() && !c.expiry.IsZero() {
		// We give a 1 minute window as a buffer
		return time.Until(c.expiry) <= time.Minute
	}
	return false
}

func extractExpiryFromJWT(token string) time.Time {
	claims := jwt.MapClaims{}
	// Ignore errors since it can be an invalid token as well
	_, _, _ = jwt.NewParser().ParseUnverified(token, claims)
	if t, _ := claims.GetExpirationTime(); t != nil {
		return t.Time
	}
	return time.Time{}
}
