package context

import (
	gocontext "context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	onepassword "github.com/1password/onepassword-sdk-go"
)

const (
	onePasswordReferencePrefix        = "op://"
	onePasswordServiceAccountTokenEnv = "OP_SERVICE_ACCOUNT_TOKEN"
	onePasswordIntegrationName        = "Flanksource Duty"
)

type onePasswordSecrets interface {
	Resolve(ctx gocontext.Context, secretReference string) (string, error)
}

type onePasswordSDKClient struct {
	client *onepassword.Client
}

func (c *onePasswordSDKClient) Resolve(ctx gocontext.Context, secretReference string) (string, error) {
	return c.client.Secrets().Resolve(ctx, secretReference)
}

type onePasswordClientFactory func(ctx gocontext.Context, token string) (onePasswordSecrets, error)

type onePasswordResolver struct {
	mu               sync.Mutex
	tokenFingerprint [sha256.Size]byte
	secrets          onePasswordSecrets
	newClient        onePasswordClientFactory
}

var defaultOnePasswordResolver = &onePasswordResolver{
	newClient: newOnePasswordSecrets,
}

func newOnePasswordSecrets(ctx gocontext.Context, token string) (onePasswordSecrets, error) {
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo(onePasswordIntegrationName, onepassword.DefaultIntegrationVersion),
	)
	if err != nil {
		return nil, err
	}
	return &onePasswordSDKClient{client: client}, nil
}

func (r *onePasswordResolver) resolve(ctx Context, token, reference string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%s is required to resolve a 1Password reference", onePasswordServiceAccountTokenEnv)
	}
	if !strings.HasPrefix(reference, onePasswordReferencePrefix) {
		return "", fmt.Errorf("invalid 1Password reference: must start with %s", onePasswordReferencePrefix)
	}

	cacheFingerprint := sha256.Sum256([]byte(token + "\x00" + reference))
	cacheID := fmt.Sprintf("onepassword/%x", cacheFingerprint)
	if value, found := envCache.Get(cacheID); found {
		return value.(string), nil
	}

	secrets, err := r.client(ctx, token)
	if err != nil {
		return "", fmt.Errorf("create 1Password client: %w", err)
	}

	value, err := secrets.Resolve(ctx, reference)
	if err != nil {
		return "", fmt.Errorf("resolve 1Password secret: %w", err)
	}

	envCache.Set(cacheID, value, ctx.Properties().Duration("envvar.cache.timeout", 5*time.Minute))
	return value, nil
}

func (r *onePasswordResolver) client(ctx gocontext.Context, token string) (onePasswordSecrets, error) {
	tokenFingerprint := sha256.Sum256([]byte(token))

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.secrets != nil && r.tokenFingerprint == tokenFingerprint {
		return r.secrets, nil
	}

	secrets, err := r.newClient(ctx, token)
	if err != nil {
		return nil, err
	}
	r.secrets = secrets
	r.tokenFingerprint = tokenFingerprint
	return secrets, nil
}

func getOnePasswordSecretFromCache(ctx Context, reference string) (string, error) {
	token, ok := os.LookupEnv(onePasswordServiceAccountTokenEnv)
	if !ok || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%s is required to resolve a 1Password reference", onePasswordServiceAccountTokenEnv)
	}
	return defaultOnePasswordResolver.resolve(ctx, token, reference)
}
