// Referenced from: https://github.com/kubernetes/client-go/blob/master/plugin/pkg/client/auth/exec/exec.go
package auth

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"time"

	"github.com/flanksource/commons/logger"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/dump"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/pkg/apis/clientauthentication/install"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/metrics"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/connrotation"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

const execInfoEnv = "KUBERNETES_EXEC_INFO"

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	install.Install(scheme)
}

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

// sometimes rate limits how often a function f() is called. Specifically, Do()
// will run the provided function f() up to threshold times every interval
// duration.
type sometimes struct {
	threshold int
	interval  time.Duration

	clock clock.Clock
	mu    sync.Mutex

	count  int       // times we have called f() in this window
	window time.Time // beginning of current window of length interval
}

func (s *sometimes) Do(f func()) {
	logger.Infof("Sometimes od called")
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()
	if s.window.IsZero() {
		s.window = now
	}

	// If we are no longer in our saved time window, then we get to reset our run
	// count back to 0 and start increasing towards the threshold again.
	if inWindow := now.Sub(s.window) < s.interval; !inWindow {
		s.window = now
		s.count = 0
	}

	// If we have not run the function more than threshold times in this current
	// time window, we get to run it now!
	if underThreshold := s.count < s.threshold; underThreshold {
		s.count++
		f()
	}
}

// GetAuthenticator returns an exec-based plugin for providing client credentials.
func GetAuthenticator(callback func() error) (*Authenticator, error) {
	return newAuthenticator(callback)
}

func newAuthenticator(callback func() error) (*Authenticator, error) {
	connTracker := connrotation.NewConnectionTracker()
	defaultDialer := connrotation.NewDialerWithTracker(
		(&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		connTracker,
	)

	a := &Authenticator{
		sometimes: &sometimes{
			threshold: 10,
			interval:  time.Hour,
			clock:     clock.RealClock{},
		},

		stdin:   os.Stdin,
		stderr:  os.Stderr,
		now:     time.Now,
		environ: os.Environ,

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
	// Set by the config
	cmd                string
	args               []string
	group              schema.GroupVersion
	env                []string
	cluster            *clientauthentication.Cluster
	provideClusterInfo bool

	// Used to avoid log spew by rate limiting install hint printing. We didn't do
	// this by interval based rate limiting alone since that way may have prevented
	// the install hint from showing up for kubectl users.
	sometimes   *sometimes
	installHint string

	// Stubbable for testing
	stdin           io.Reader
	stderr          io.Writer
	interactiveFunc func() (bool, error)
	now             func() time.Time
	environ         func() []string

	callback func() error

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

// UpdateTransportConfig updates the transport.Config to use credentials
// returned by the plugin.
func (a *Authenticator) UpdateTransportConfig(c *transport.Config) error {
	// If a bearer token is present in the request - avoid the GetCert callback when
	// setting up the transport, as that triggers the exec action if the server is
	// also configured to allow client certificates for authentication. For requests
	// like "kubectl get --token (token) pods" we should assume the intention is to
	// use the provided token for authentication. The same can be said for when the
	// user specifies basic auth or cert auth.
	logger.Infof("000- AAAA")
	if c.HasTokenAuth() || c.HasBasicAuth() || c.HasCertAuth() {
		logger.Infof("AAAA")
		return nil
	}

	logger.Infof("111- AAAA")
	c.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &YroundTripper{a, rt}
	})

	logger.Infof("222- AAAA")
	if c.HasCertCallback() {
		logger.Infof("This err")
		//return errors.New("can't add TLS certificate callback: transport.Config.TLS.GetCert already set")
	}
	c.TLS.GetCertHolder = a.getCert // comparable for TLS config caching

	logger.Infof("333- AAAA")
	if c.DialHolder != nil {
		logger.Infof("444- AAAA")
		if c.DialHolder.Dial == nil {
			return errors.New("invalid transport.Config.DialHolder: wrapped Dial function is nil")
		}

		logger.Infof("555- AAAA")
		// if c has a custom dialer, we have to wrap it
		// TLS config caching is not supported for this config
		d := connrotation.NewDialerWithTracker(c.DialHolder.Dial, a.connTracker)
		c.DialHolder = &transport.DialHolder{Dial: d.DialContext}
	} else {
		c.DialHolder = a.dial // comparable for TLS config caching
	}

	logger.Infof("222- AAAA")
	return nil
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
	if req.Header.Get("Authorization") != "" {
		logger.Infof("IN ROUND TRIP auth header set is %s", req.Header.Get("Authorization"))
		return r.base.RoundTrip(req)
	}

	logger.Infof("IN ROUND TRIP")
	creds, err := r.a.getCreds()
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %v", err)
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
	logger.Infof("credsExpired called")
	if a.exp.IsZero() {
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
	return a.callback()
}

func (a *Authenticator) getCreds() (*credentials, error) {
	logger.Infof("get creds called")
	a.mu.Lock()
	defer a.mu.Unlock()

	//if a.cachedCreds != nil && !a.credsExpired() {
	//return a.cachedCreds, nil
	//}

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
	if err := a.callback(); err != nil {
		return fmt.Errorf("error calling callback: %w", err)
	}
	if true {
		return nil
	}
	interactive, err := a.interactiveFunc()
	if err != nil {
		return fmt.Errorf("exec plugin cannot support interactive mode: %w", err)
	}

	cred := &clientauthentication.ExecCredential{
		Spec: clientauthentication.ExecCredentialSpec{
			Interactive: interactive,
		},
	}
	if a.provideClusterInfo {
		cred.Spec.Cluster = a.cluster
	}

	env := append(a.environ(), a.env...)
	data, err := runtime.Encode(codecs.LegacyCodec(a.group), cred)
	if err != nil {
		return fmt.Errorf("encode ExecCredentials: %v", err)
	}
	env = append(env, fmt.Sprintf("%s=%s", execInfoEnv, data))

	stdout := &bytes.Buffer{}
	cmd := exec.Command(a.cmd, a.args...)
	cmd.Env = env
	cmd.Stderr = a.stderr
	cmd.Stdout = stdout
	if interactive {
		cmd.Stdin = a.stdin
	}

	err = cmd.Run()
	if err != nil {
		return err
	}

	_, gvk, err := codecs.UniversalDecoder(a.group).Decode(stdout.Bytes(), nil, cred)
	if err != nil {
		return fmt.Errorf("decoding stdout: %v", err)
	}
	if gvk.Group != a.group.Group || gvk.Version != a.group.Version {
		return fmt.Errorf("exec plugin is configured to use API version %s, plugin returned version %s",
			a.group, schema.GroupVersion{Group: gvk.Group, Version: gvk.Version})
	}

	if cred.Status == nil {
		return fmt.Errorf("exec plugin didn't return a status field")
	}
	if cred.Status.Token == "" && cred.Status.ClientCertificateData == "" && cred.Status.ClientKeyData == "" {
		return fmt.Errorf("exec plugin didn't return a token or cert/key pair")
	}
	if (cred.Status.ClientCertificateData == "") != (cred.Status.ClientKeyData == "") {
		return fmt.Errorf("exec plugin returned only certificate or key, not both")
	}

	if cred.Status.ExpirationTimestamp != nil {
		a.exp = cred.Status.ExpirationTimestamp.Time
	} else {
		a.exp = time.Time{}
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
