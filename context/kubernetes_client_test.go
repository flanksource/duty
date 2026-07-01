package context_test

import (
	stdcontext "context"
	"encoding/base64"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/connection"
	dutycontext "github.com/flanksource/duty/context"
	dutykubernetes "github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/pkg/kube/auth"
	"github.com/flanksource/duty/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// Use unique secret names because the Kubernetes client and env var caches are package globals.
var kubernetesConnectionCacheCollisionTestID atomic.Uint64

func TestKubernetesConnectionCacheCollision(t *testing.T) {
	t.Run("basic auth", func(t *testing.T) {
		providerClient, consumerClient := runKubernetesConnectionCacheCollisionTest(
			t,
			uniqueKubeconfigSecretName("basic-auth"),
			basicAuthKubeconfig("provider", providerHost),
			basicAuthKubeconfig("consumer", consumerHost),
		)

		assertDistinctKubernetesClients(t, providerClient, consumerClient)
	})

	t.Run("bearer token", func(t *testing.T) {
		providerToken := jwtToken("provider")
		consumerToken := jwtToken("consumer")

		providerClient, consumerClient := runKubernetesConnectionCacheCollisionTest(
			t,
			uniqueKubeconfigSecretName("bearer-token"),
			bearerTokenKubeconfig("provider", providerHost, providerToken),
			bearerTokenKubeconfig("consumer", consumerHost, consumerToken),
		)

		assertDistinctKubernetesClients(t, providerClient, consumerClient)
		assertAuthCallbackToken(t, providerClient, providerToken)
		assertAuthCallbackToken(t, consumerClient, consumerToken)
	})
}

const (
	secretKey    = "config"
	providerNS   = "provider"
	consumerNS   = "consumer"
	providerHost = "https://provider.example.invalid"
	consumerHost = "https://consumer.example.invalid"
)

func runKubernetesConnectionCacheCollisionTest(t *testing.T, secretName, providerConfig, consumerConfig string) (*dutykubernetes.Client, *dutykubernetes.Client) {
	t.Helper()
	resetKubernetesClientTestState(t)
	t.Cleanup(func() {
		resetKubernetesClientTestState(t)
	})

	local := dutykubernetes.NewKubeClient(
		logger.GetLogger("test"),
		fake.NewSimpleClientset(
			kubeconfigSecret(providerNS, secretName, secretKey, providerConfig),
			kubeconfigSecret(consumerNS, secretName, secretKey, consumerConfig),
		),
		&rest.Config{},
	)

	base := dutycontext.New().WithLocalKubernetes(local)

	providerClient, err := base.
		WithNamespace(providerNS).
		WithKubernetes(kubeconfigSecretRefConnection(secretName, secretKey)).
		Kubernetes()
	if err != nil {
		t.Fatalf("provider Kubernetes(): %v", err)
	}
	consumerClient, err := base.
		WithNamespace(consumerNS).
		WithKubernetes(kubeconfigSecretRefConnection(secretName, secretKey)).
		Kubernetes()
	if err != nil {
		t.Fatalf("consumer Kubernetes(): %v", err)
	}
	return providerClient, consumerClient
}

func assertDistinctKubernetesClients(t *testing.T, providerClient, consumerClient *dutykubernetes.Client) {
	t.Helper()
	if providerClient == consumerClient {
		t.Fatalf("provider and consumer reused the same Kubernetes client")
	}
	if got := providerClient.RestConfig().Host; got != providerHost {
		t.Fatalf("provider host: got %q, want %q", got, providerHost)
	}
	if got := consumerClient.RestConfig().Host; got != consumerHost {
		t.Fatalf("consumer host: got %q, want %q", got, consumerHost)
	}
}

func assertAuthCallbackToken(t *testing.T, client *dutykubernetes.Client, want string) {
	t.Helper()

	restConfig := client.RestConfig()
	if restConfig.AuthProvider == nil {
		t.Fatalf("expected rest config to use duty auth provider")
	}

	callbackKey := restConfig.AuthProvider.Config["conn"]
	if callbackKey == "" {
		t.Fatalf("expected duty auth provider conn key")
	}

	callback, err := auth.AuthKubernetesCallbackCache.Get(stdcontext.Background(), callbackKey)
	if err != nil {
		t.Fatalf("get auth callback %q: %v", callbackKey, err)
	}

	refreshed, err := callback()
	if err != nil {
		t.Fatalf("refresh auth callback %q: %v", callbackKey, err)
	}
	if got := refreshed.BearerToken; got != want {
		t.Fatalf("auth callback token: got %q, want %q", got, want)
	}
}

func resetKubernetesClientTestState(t *testing.T) {
	t.Helper()

	dutycontext.New().WithLocalKubernetes(nil)
	if err := auth.AuthKubernetesCallbackCache.Clear(stdcontext.Background()); err != nil {
		t.Fatalf("clear auth callback cache: %v", err)
	}
}

func uniqueKubeconfigSecretName(prefix string) string {
	return fmt.Sprintf("config-%s-%d", prefix, kubernetesConnectionCacheCollisionTestID.Add(1))
}

func kubeconfigSecretRefConnection(secretName, secretKey string) connection.KubernetesConnection {
	return connection.KubernetesConnection{
		KubeconfigConnection: connection.KubeconfigConnection{
			Kubeconfig: &types.EnvVar{
				ValueFrom: &types.EnvVarSource{
					SecretKeyRef: &types.SecretKeySelector{
						LocalObjectReference: types.LocalObjectReference{Name: secretName},
						Key:                  secretKey,
					},
				},
			},
		},
	}
}

func kubeconfigSecret(namespace, name, key, value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			key: []byte(value),
		},
	}
}

func basicAuthKubeconfig(name, server string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %s
  cluster:
    server: %s
    insecure-skip-tls-verify: true
users:
- name: %s
  user:
    username: %s
    password: password
contexts:
- name: %s
  context:
    cluster: %s
    user: %s
current-context: %s
`, name, server, name, name, name, name, name, name)
}

func bearerTokenKubeconfig(name, server, token string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %s
  cluster:
    server: %s
    insecure-skip-tls-verify: true
users:
- name: %s
  user:
    token: %q
contexts:
- name: %s
  context:
    cluster: %s
    user: %s
current-context: %s
`, name, server, name, token, name, name, name, name)
}

func jwtToken(subject string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"sub":%q,"exp":%d}`, subject, time.Now().Add(time.Hour).Unix())))
	return header + "." + payload + "."
}
