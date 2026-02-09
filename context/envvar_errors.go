package context

import (
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var ErrSecretLookupRateLimited = errors.New("secret lookup rate limited")

func newSecretLookupRateLimitedError(namespace, name, key string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: secret(%s/%s).%s", ErrSecretLookupRateLimited, namespace, name, key)
	}
	return fmt.Errorf("%w: secret(%s/%s).%s: %v", ErrSecretLookupRateLimited, namespace, name, key, cause)
}

func IsSecretLookupRateLimited(err error) bool {
	return errors.Is(err, ErrSecretLookupRateLimited)
}

func isSecretLookupRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	if apierrors.IsTooManyRequests(err) {
		return true
	}

	// client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	// doesn't return a typed error, so we need to do a string lookup.
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "client rate limiter wait returned an error")
}
