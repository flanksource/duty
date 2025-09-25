package context

import (
	"fmt"
	"time"

	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/flanksource/commons/logger"
	dutyKubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/pkg/kube/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"
)

type KubernetesClient struct {
	*dutyKubernetes.Client
	Connection KubernetesConnection
	expiry     time.Time
	logger     logger.Logger
}

var defaultExpiry = 15 * time.Minute

func authProvider(clusterAddress string, config map[string]string, persister rest.AuthProviderConfigPersister) (rest.AuthProvider, error) {
	connHash := config["conn"]
	if connHash == "" {
		return nil, fmt.Errorf("key[conn] with connection hash not set")
	}
	ap, err := auth.GetAuthenticator(connHash)
	return ap, err
}

func NewKubernetesClient(ctx Context, conn KubernetesConnection) (*KubernetesClient, error) {
	c, rc, err := conn.Populate(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("error refreshing kubernetes client: %w", err)
	}
	log := logger.GetLogger("k8s." + conn.String())
	client := &KubernetesClient{
		Client:     dutyKubernetes.NewKubeClient(log, c, rc),
		Connection: conn,
		logger:     logger.GetLogger("k8s"),
	}

	if client.logger.IsLevelEnabled(logger.Trace4) {
		client.logger.V(logger.Trace1).Infof(logger.Stacktrace())
	}

	client.SetExpiry(defaultExpiry)

	connHash := conn.Hash()
	if rc.ExecProvider == nil && rc.BearerToken != "" {
		refreshCallback := func() (*rest.Config, error) {
			ctx.Counter("kubernetes_auth_plugin_refreshed", "connection", connHash).Add(1)
			rc, err := client.Refresh(ctx)
			return rc, err
		}
		rc.BearerToken = ""
		if err := auth.AuthKubernetesCallbackCache.Set(ctx, connHash, refreshCallback); err != nil {
			return nil, err
		}
		rc.AuthProvider = &clientcmdapi.AuthProviderConfig{
			Name:   "duty",
			Config: map[string]string{"conn": conn.Hash()},
		}
	}

	client.SetLogger(logger.GetLogger("k8s." + dutyKubernetes.GetClusterName(rc)))

	client.logger.V(3).Infof("created new client with expiry: %s", client.expiry.Format(time.RFC3339))
	return client, nil
}

func (c *KubernetesClient) SetLogger(log logger.Logger) {
	c.logger = log
	c.Client.SetLogger(log)
}

func (c *KubernetesClient) SetExpiry(def time.Duration) {
	// Try parsing BearerToken as JWT and extract expiry
	if expiry := extractExpiryFromJWT(lo.FromPtr(c.Config).BearerToken); !expiry.IsZero() {
		c.expiry = expiry
	} else {
		c.expiry = time.Now().Add(def)
	}
}

func (c *KubernetesClient) Refresh(ctx Context) (*rest.Config, error) {
	if !c.HasExpired() && (c.Config.AuthProvider == nil || c.Config.BearerToken != "") {
		return c.RestConfig(), nil
	}
	client, rc, err := c.Connection.Populate(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("error refreshing kubernetes client: %w", err)
	}

	// Update rest config in place for easy reuse
	c.Config.Host = rc.Host
	c.Config.TLSClientConfig = rc.TLSClientConfig
	c.Config.BearerTokenFile = rc.BearerTokenFile
	c.Config.Username = rc.Username

	if c.Config.BearerToken != rc.BearerToken || c.Config.Password != rc.Password {
		c.Config.BearerToken = rc.BearerToken
		c.Config.Password = rc.Password
		c.Client.Reset()
	}

	c.Client.Interface = client
	c.SetExpiry(defaultExpiry)
	c.logger.V(5).Infof("token refreshed, expires at %s", c.expiry.Format(time.RFC3339))
	return rc, nil
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

func init() {
	_ = rest.RegisterAuthProviderPlugin("duty", authProvider)
}
