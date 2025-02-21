package context

import (
	"fmt"
	"time"

	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/golang-jwt/jwt/v5"
)

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
}

func (c *KubernetesClient) SetExpiry(d time.Duration) {
	c.expiry = time.Now().Add(d)
}
func (c *KubernetesClient) ExpireAt(t time.Time) {
	if !t.IsZero() {
		c.expiry = t
	}
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

	// Try parsing BearerToken as JWT and extract expiry
	if expiry := extractExpiryFromJWT(c.Config.BearerToken); !expiry.IsZero() {
		c.ExpireAt(expiry)
	} else {
		c.SetExpiry(d)
	}

	return nil
}

func (c KubernetesClient) HasExpired() bool {
	if c.Connection.CanExpire() {
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
