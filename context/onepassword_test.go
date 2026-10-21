package context

import (
	gocontext "context"
	"errors"
	"sync"

	"github.com/flanksource/duty/types"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeOnePasswordSecrets struct {
	mu      sync.Mutex
	value   string
	err     error
	resolve int
}

func (f *fakeOnePasswordSecrets) Resolve(_ gocontext.Context, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolve++
	return f.value, f.err
}

func (f *fakeOnePasswordSecrets) ResolveCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resolve
}

var _ = ginkgo.Describe("1Password EnvVar", func() {
	const reference = "op://example-vault/example-item/password"

	ginkgo.BeforeEach(func() {
		envCache.Flush()
		ginkgo.GinkgoT().Setenv(onePasswordServiceAccountTokenEnv, "account-token")
	})

	ginkgo.AfterEach(func() {
		envCache.Flush()
	})

	ginkgo.It("resolves and caches a value-level secret reference", func() {
		secrets := &fakeOnePasswordSecrets{value: "resolved-secret"}
		factoryCalls := 0
		resolver := &onePasswordResolver{
			newClient: func(_ gocontext.Context, token string) (onePasswordSecrets, error) {
				factoryCalls++
				Expect(token).To(Equal("account-token"))
				return secrets, nil
			},
		}
		originalResolver := defaultOnePasswordResolver
		defaultOnePasswordResolver = resolver
		ginkgo.DeferCleanup(func() {
			defaultOnePasswordResolver = originalResolver
		})

		ctx := NewContext(gocontext.Background())
		env := types.EnvVar{Name: "PASSWORD", ValueStatic: reference}
		first, firstErr := GetEnvValueFromCache(ctx, env, "")
		second, secondErr := GetEnvValueFromCache(ctx, env, "")

		Expect(firstErr).NotTo(HaveOccurred())
		Expect(secondErr).NotTo(HaveOccurred())
		Expect([]string{first, second}).To(Equal([]string{"resolved-secret", "resolved-secret"}))
		Expect(factoryCalls).To(Equal(1))
		Expect(secrets.ResolveCount()).To(Equal(1))
	})

	ginkgo.It("leaves literal values unchanged", func() {
		factoryCalls := 0
		originalResolver := defaultOnePasswordResolver
		defaultOnePasswordResolver = &onePasswordResolver{
			newClient: func(_ gocontext.Context, _ string) (onePasswordSecrets, error) {
				factoryCalls++
				return &fakeOnePasswordSecrets{}, nil
			},
		}
		ginkgo.DeferCleanup(func() {
			defaultOnePasswordResolver = originalResolver
		})

		value, err := GetEnvValueFromCache(
			NewContext(gocontext.Background()),
			types.EnvVar{Name: "USERNAME", ValueStatic: "literal-value"},
			"",
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal("literal-value"))
		Expect(factoryCalls).To(BeZero())
	})

	ginkgo.It("creates a new client and cache entry after token rotation", func() {
		factoryTokens := make([]string, 0, 2)
		resolver := &onePasswordResolver{
			newClient: func(_ gocontext.Context, token string) (onePasswordSecrets, error) {
				factoryTokens = append(factoryTokens, token)
				return &fakeOnePasswordSecrets{value: "secret-for-" + token}, nil
			},
		}
		ctx := NewContext(gocontext.Background())

		first, firstErr := resolver.resolve(ctx, "account-token-a", reference)
		second, secondErr := resolver.resolve(ctx, "account-token-b", reference)

		Expect(firstErr).NotTo(HaveOccurred())
		Expect(secondErr).NotTo(HaveOccurred())
		Expect([]string{first, second}).To(Equal([]string{"secret-for-account-token-a", "secret-for-account-token-b"}))
		Expect(factoryTokens).To(Equal([]string{"account-token-a", "account-token-b"}))
	})

	ginkgo.It("reuses one client across concurrent references", func() {
		secrets := &fakeOnePasswordSecrets{value: "resolved-secret"}
		factoryCalls := 0
		resolver := &onePasswordResolver{
			newClient: func(_ gocontext.Context, _ string) (onePasswordSecrets, error) {
				factoryCalls++
				return secrets, nil
			},
		}
		ctx := NewContext(gocontext.Background())
		references := []string{
			"op://example-vault/example-item/username",
			"op://example-vault/example-item/password",
			"op://example-vault/example-item/token",
		}
		errs := make(chan error, len(references))
		var waitGroup sync.WaitGroup
		for _, item := range references {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				_, err := resolver.resolve(ctx, "account-token", item)
				errs <- err
			}()
		}
		waitGroup.Wait()
		close(errs)

		for err := range errs {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(factoryCalls).To(Equal(1))
		Expect(secrets.ResolveCount()).To(Equal(len(references)))
	})

	ginkgo.It("fails when the service account token is empty", func() {
		ginkgo.GinkgoT().Setenv(onePasswordServiceAccountTokenEnv, "")

		_, err := getOnePasswordSecretFromCache(NewContext(gocontext.Background()), reference)

		Expect(err).To(MatchError(ContainSubstring(onePasswordServiceAccountTokenEnv)))
	})

	ginkgo.DescribeTable("returns contextual SDK errors",
		func(factoryError, resolveError error, expected string) {
			resolver := &onePasswordResolver{
				newClient: func(_ gocontext.Context, _ string) (onePasswordSecrets, error) {
					return &fakeOnePasswordSecrets{err: resolveError}, factoryError
				},
			}

			_, err := resolver.resolve(NewContext(gocontext.Background()), "account-token", reference)

			Expect(err).To(MatchError(ContainSubstring(expected)))
		},
		ginkgo.Entry("client initialization", errors.New("authentication rejected"), nil, "create 1Password client"),
		ginkgo.Entry("secret resolution", nil, errors.New("reference denied"), "resolve 1Password secret"),
	)
})
