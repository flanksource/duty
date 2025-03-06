package context

import (
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
	logger     logger.Logger
}

var defaultExpiry = 1 * time.Minute

func NewKubernetesClient(ctx Context, conn KubernetesConnection) (*KubernetesClient, error) {
	c, rc, err := conn.Populate(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("error refreshing kubernetes client: %w", err)
	}
	client := &KubernetesClient{
		Client:     dutyKubernetes.NewKubeClient(c, rc),
		Connection: conn,
		logger:     logger.GetLogger("k8s").Named(conn.String()),
	}

	if client.logger.IsLevelEnabled(logger.Trace1) {
		client.logger.V(logger.Trace1).Infof(logger.Stacktrace())
	}

	client.SetExpiry(defaultExpiry)
	client.logger.Infof("created new client for %s with expiry: %s", lo.FromPtr(rc).Host, client.expiry)
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
		c.logger.Infof("Skipping refresh, client has not expired %s", c.Config.Host)
		return nil
	}
	client, rc, err := c.Connection.Populate(ctx, true)
	if err != nil {
		return fmt.Errorf("error refreshing kubernetes client: %w", err)
	}

	doRefresh := false
	// Update rest config in place for easy reuse
	c.Config.Host = rc.Host
	c.Config.TLSClientConfig = rc.TLSClientConfig
	if c.Config.BearerToken != rc.BearerToken {
		logger.Infof("=== BEARER TOKEN UPDATED FOR host: %s", c.Config.Host)
		doRefresh = true
		c.Config.BearerToken = rc.BearerToken
	}
	if c.Config.Password != rc.Password {
		doRefresh = true
	}

	c.Config.BearerTokenFile = rc.BearerTokenFile
	c.Config.Username = rc.Username
	c.Config.Password = rc.Password

	logger.Infof("Host %s", rc.Host)
	logger.Infof("BearerToken %+v", rc.BearerToken)
	logger.Infof("Address of bt in Refresh is %p", &c.Client.Config.BearerToken)

	if doRefresh {
		logger.Infof("Refreshing client for host %s", rc.Host)
		c.Client.Interface = client
		c.Client.Refresh()
	}

	c.SetExpiry(defaultExpiry)
	c.logger.Debugf("Refreshed %s, expires at %s", rc.Host, c.expiry)
	return nil
}

func (c KubernetesClient) HasExpired() bool {
	if c.Connection.CanExpire() && !c.expiry.IsZero() {
		// We give a 1 minute window as a buffer
		g := time.Until(c.expiry) <= time.Minute
		return g
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
