// Referenced from: https://github.com/kubernetes/client-go/blob/master/plugin/pkg/client/auth/exec/exec.go
package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/flanksource/commons/logger"
	dutyCache "github.com/flanksource/duty/cache"
	"k8s.io/apimachinery/pkg/util/dump"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/metrics"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/connrotation"
	"k8s.io/klog/v2"
)

var (
	// Since transports can be constantly re-initialized by programs like kubectl,
	// keep a cache of initialized authenticators keyed by a hash of their config.
	globalCache = newCache()
)

func newCache() *cache {
	return &cache{m: make(map[string]*Authenticator)}
}

func cacheKey(conf *api.ExecConfig, cluster *clientauthentication.Cluster) string {
	key := struct {
		conf    *api.ExecConfig
		cluster *clientauthentication.Cluster
	}{
		conf:    conf,
		cluster: cluster,
	}
	return dump.Pretty(key)
}

type cache struct {
	mu sync.Mutex
	m  map[string]*Authenticator
}

func (c *cache) get(s string) (*Authenticator, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a, ok := c.m[s]
	return a, ok
}

// put inserts an authenticator into the cache. If an authenticator is already
// associated with the key, the first one is returned instead.
func (c *cache) put(s string, a *Authenticator) *Authenticator {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.m[s]
	if ok {
		return existing
	}
	c.m[s] = a
	return a
}

// GetAuthenticator returns an exec-based plugin for providing client credentials.
func GetAuthenticator(connHash string) (*Authenticator, error) {
	return newAuthenticator(connHash)
}

type CB func() (*rest.Config, error)

var K8sclientcache2 = dutyCache.NewCache[CB]("k8s-client-cache", 24*time.Hour)
var sm sync.Map

func newAuthenticator(connHash string) (*Authenticator, error) {
	connTracker := connrotation.NewConnectionTracker()
	defaultDialer := connrotation.NewDialerWithTracker(
		(&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		connTracker,
	)

	logger.Infof("newAuthenticator for conn: %s", connHash)
	callback, err := K8sclientcache2.Get(context.Background(), connHash)
	if err != nil {
		return nil, err
	}
	a := &Authenticator{
		now: time.Now,

		connTracker: connTracker,
		callback:    callback,
	}

	// these functions are made comparable and stored in the cache so that repeated clientset
	// construction with the same rest.Config results in a single TLS cache and Authenticator
	a.getCert = &transport.GetCertHolder{GetCert: a.cert}
	a.dial = &transport.DialHolder{Dial: defaultDialer.DialContext}

	return a, nil
}

// Authenticator is a client credential provider that rotates credentials by executing a plugin.
// The plugin input and output are defined by the API group client.authentication.k8s.io.
type Authenticator struct {
	// Stubbable for testing
	now func() time.Time

	callback CB

	// connTracker tracks all connections opened that we need to close when rotating a client certificate
	connTracker *connrotation.ConnectionTracker

	// Cached results.
	//
	// The mutex also guards calling the plugin. Since the plugin could be
	// interactive we want to make sure it's only called once.
	mu          sync.Mutex
	cachedCreds *credentials
	exp         time.Time

	// getCert makes Authenticator.cert comparable to support TLS config caching
	getCert *transport.GetCertHolder
	// dial is used for clients which do not specify a custom dialer
	// it is comparable to support TLS config caching
	dial *transport.DialHolder
}

type credentials struct {
	token string           `datapolicy:"token"`
	cert  *tls.Certificate `datapolicy:"secret-key"`
}

var _ utilnet.RoundTripperWrapper = &YroundTripper{}

type YroundTripper struct {
	a    *Authenticator
	base http.RoundTripper
}

func (r *YroundTripper) WrappedRoundTripper() http.RoundTripper {
	logger.Infof("IN WRAPPED ROUND TRIP")
	return r.base
}

func (r *YroundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// If a user has already set credentials, use that. This makes commands like
	// "kubectl get --token (token) pods" work.
	//if req.Header.Get("Authorization") != "" {
	//return r.base.RoundTrip(req)
	//}

	creds, err := r.a.getCreds()
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %v", err)
	}

	if req.Header.Get("Authorization") != "Bearer "+creds.token {
		//logger.Infof("Mismatch for set header[%s] and creds.token=%s", req.Header.Get("Authorization"), "Bearer "+creds.token)
		logger.Infof("Auth header mismatch, doing refreshCredsLocked")
		if err := r.a.refreshCredsLocked(); err != nil {
			return nil, fmt.Errorf("error refreshing creds: %w", err)
		}
	}

	if creds.token != "" {
		req.Header.Set("Authorization", "Bearer "+creds.token)
	}

	res, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		logger.Infof("Unauth")
		if err := r.a.maybeRefreshCreds(creds); err != nil {
			klog.Errorf("refreshing credentials: %v", err)
		}
	}
	return res, nil
}

func (a *Authenticator) credsExpired() bool {
	if a.exp.IsZero() {
		logger.Infof("credsExpired called 0 expiry")
		return false
	}
	return a.now().After(a.exp)
}

func (a *Authenticator) cert() (*tls.Certificate, error) {
	logger.Infof("CERT CALLED")
	creds, err := a.getCreds()
	if err != nil {
		return nil, err
	}
	return creds.cert, nil
}

func (a *Authenticator) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	logger.Infof("WrapTransport from a")
	return &YroundTripper{a, rt}
}

func (a *Authenticator) Login() error {
	logger.Infof("Login")
	_, err := a.callback()
	return err
}

func (a *Authenticator) getCreds() (*credentials, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cachedCreds != nil && !a.credsExpired() {
		return a.cachedCreds, nil
	}

	logger.Infof("cached creds not returned")
	if err := a.refreshCredsLocked(); err != nil {
		return nil, err
	}

	return a.cachedCreds, nil
}

// maybeRefreshCreds executes the plugin to force a rotation of the
// credentials, unless they were rotated already.
func (a *Authenticator) maybeRefreshCreds(creds *credentials) error {
	logger.Infof("maybe frefresh creds called")
	a.mu.Lock()
	defer a.mu.Unlock()

	// Since we're not making a new pointer to a.cachedCreds in getCreds, no
	// need to do deep comparison.
	if creds != a.cachedCreds {
		// Credentials already rotated.
		return nil
	}

	return a.refreshCredsLocked()
}

// refreshCredsLocked executes the plugin and reads the credentials from
// stdout. It must be called while holding the Authenticator's mutex.
func (a *Authenticator) refreshCredsLocked() error {

	logger.Infof("refreshCredsLocked called")
	// Call callback
	rc, err := a.callback()
	if err != nil {
		return fmt.Errorf("error calling callback: %w", err)
	}
	logger.Infof("GOT RC token=%s rc.KeyData=%s rc.CertData=%s", rc.BearerToken, string(rc.KeyData), string(rc.CertData))

	cred := &clientauthentication.ExecCredential{
		Status: &clientauthentication.ExecCredentialStatus{
			Token:                 rc.BearerToken,
			ClientKeyData:         string(rc.KeyData),
			ClientCertificateData: string(rc.CertData),
		},
	}

	if cred.Status.Token == "" && cred.Status.ClientCertificateData == "" && cred.Status.ClientKeyData == "" {
		return fmt.Errorf("exec plugin didn't return a token or cert/key pair")
	}
	if (cred.Status.ClientCertificateData == "") != (cred.Status.ClientKeyData == "") {
		return fmt.Errorf("plugin returned only certificate or key, not both")
	}

	if cred.Status.ExpirationTimestamp != nil {
		a.exp = cred.Status.ExpirationTimestamp.Time
	} else {
		a.exp = time.Now().Add(15 * time.Minute)
	}

	newCreds := &credentials{
		token: cred.Status.Token,
	}
	if cred.Status.ClientKeyData != "" && cred.Status.ClientCertificateData != "" {
		cert, err := tls.X509KeyPair([]byte(cred.Status.ClientCertificateData), []byte(cred.Status.ClientKeyData))
		if err != nil {
			return fmt.Errorf("failed parsing client key/certificate: %v", err)
		}

		// Leaf is initialized to be nil:
		//  https://golang.org/pkg/crypto/tls/#X509KeyPair
		// Leaf certificate is the first certificate:
		//  https://golang.org/pkg/crypto/tls/#Certificate
		// Populating leaf is useful for quickly accessing the underlying x509
		// certificate values.
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("failed parsing client leaf certificate: %v", err)
		}
		newCreds.cert = &cert
	}

	oldCreds := a.cachedCreds
	a.cachedCreds = newCreds
	// Only close all connections when TLS cert rotates. Token rotation doesn't
	// need the extra noise.
	if oldCreds != nil && !reflect.DeepEqual(oldCreds.cert, a.cachedCreds.cert) {
		// Can be nil if the exec auth plugin only returned token auth.
		if oldCreds.cert != nil && oldCreds.cert.Leaf != nil {
			metrics.ClientCertRotationAge.Observe(time.Since(oldCreds.cert.Leaf.NotBefore))
		}
		a.connTracker.CloseAll()
	}

	return nil
}
