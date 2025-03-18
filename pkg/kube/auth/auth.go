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

	dutyCache "github.com/flanksource/duty/cache"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/pkg/apis/clientauthentication"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/metrics"
	"k8s.io/client-go/transport"
	"k8s.io/client-go/util/connrotation"
	"k8s.io/klog/v2"
)

// GetAuthenticator returns an exec-based plugin for providing client credentials.
func GetAuthenticator(connHash string) (*Authenticator, error) {
	return newAuthenticator(connHash)
}

type CallbackFunc func() (*rest.Config, error)

var AuthKubernetesCallbackCache = dutyCache.NewCache[CallbackFunc]("k8s-cb-cache", 0)

func newAuthenticator(connHash string) (*Authenticator, error) {
	connTracker := connrotation.NewConnectionTracker()
	defaultDialer := connrotation.NewDialerWithTracker(
		(&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		connTracker,
	)

	callback, err := AuthKubernetesCallbackCache.Get(context.Background(), connHash)
	if err != nil {
		return nil, err
	}
	a := &Authenticator{
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
	callback CallbackFunc

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

	// This is happening very often, where does the Header get originally set?
	// It gets set in the transport.HTTPWrappersForConfig function, see how rest/transport.go calls it
	// Commenting auth matching for now, and we'll always set the token to what we want and ignore the one which is set
	/*
		if req.Header.Get("Authorization") != "Bearer "+creds.token {
			//logger.Infof("Mismatch for set header[%s] and creds.token=%s", req.Header.Get("Authorization"), "Bearer "+creds.token)
			logger.Infof("Auth header mismatch, doing refreshCredsLocked")
			if err := r.a.refreshCredsLocked(); err != nil {
				return nil, fmt.Errorf("error refreshing creds: %w", err)
			}
		}*/

	if creds.token != "" {
		req.Header.Set("Authorization", "Bearer "+creds.token)
	}

	res, err := r.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusUnauthorized {
		if err := r.a.maybeRefreshCreds(creds); err != nil {
			klog.Errorf("refreshing credentials: %v", err)
		}
	}
	return res, nil
}

func (a *Authenticator) credsExpired() bool {
	if a.exp.IsZero() {
		return false
	}
	return time.Now().After(a.exp)
}

func (a *Authenticator) cert() (*tls.Certificate, error) {
	creds, err := a.getCreds()
	if err != nil {
		return nil, err
	}
	return creds.cert, nil
}

func (a *Authenticator) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &YroundTripper{a, rt}
}

func (a *Authenticator) Login() error {
	_, err := a.callback()
	return err
}

func (a *Authenticator) getCreds() (*credentials, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cachedCreds != nil && !a.credsExpired() {
		return a.cachedCreds, nil
	}

	if err := a.refreshCredsLocked(); err != nil {
		return nil, err
	}

	return a.cachedCreds, nil
}

// maybeRefreshCreds executes the plugin to force a rotation of the
// credentials, unless they were rotated already.
func (a *Authenticator) maybeRefreshCreds(creds *credentials) error {
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
	// Call callback
	rc, err := a.callback()
	if err != nil {
		return fmt.Errorf("error calling callback: %w", err)
	}

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
