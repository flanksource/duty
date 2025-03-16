package context

import (
	"fmt"
	//"net/http"
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

func fact(clusterAddress string, config map[string]string, persister rest.AuthProviderConfigPersister) (rest.AuthProvider, error) {
	connHash := config["conn"]
	ap, err := auth.GetAuthenticator(connHash)
	return ap, err
}

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

	if rc.ExecProvider == nil {
		cbWrapper := func() (*rest.Config, error) {
			rc, err := client.Refresh(ctx)
			return rc, err
		}
		rc.BearerToken = ""
		rc.Password = ""
		logger.Infof("rc beaer token empty addr %p", rc)
		if err := auth.K8sCB.Set(ctx, conn.Hash(), cbWrapper); err != nil {
			return nil, err
		}
		rc.AuthProvider = &clientcmdapi.AuthProviderConfig{
			Name:   "duty",
			Config: map[string]string{"conn": conn.Hash()},
		}
	}

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

func (c *KubernetesClient) Refresh(ctx Context) (*rest.Config, error) {
	if c.Config.BearerToken != "" && !c.HasExpired() {
		c.logger.Tracef("Skipping refresh, client has not expired for host:%s", c.Config.Host)
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
	c.logger.Debugf("Refreshed %s, expires at %s", rc.Host, c.expiry)
	return c.Config, nil
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
	rest.RegisterAuthProviderPlugin("duty", fact)
}
