package context

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RaveNoX/go-jsonmerge"
	"github.com/ohler55/ojg/jp"

	"github.com/flanksource/duty/types"
	"github.com/patrickmn/go-cache"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create a cache with a default expiration time of 5 minutes, and which
// purges expired items every 10 minutes
var envCache = cache.New(5*time.Minute, 10*time.Minute)

const helmSecretType = "helm.sh/release.v1"

func GetEnvValueFromCache(ctx Context, input types.EnvVar, namespace string) (string, error) {
	if namespace == "" {
		namespace = ctx.GetNamespace()
	}
	if input.ValueFrom == nil {
		return input.ValueStatic, nil
	}
	if input.ValueFrom.SecretKeyRef != nil {
		return GetSecretFromCache(ctx, namespace, input.ValueFrom.SecretKeyRef.Name, input.ValueFrom.SecretKeyRef.Key)
	}
	if input.ValueFrom.ConfigMapKeyRef != nil {
		return GetConfigMapFromCache(ctx, namespace, input.ValueFrom.ConfigMapKeyRef.Name, input.ValueFrom.ConfigMapKeyRef.Key)
	}
	if input.ValueFrom.HelmRef != nil {
		return GetHelmValueFromCache(ctx, namespace, input.ValueFrom.HelmRef.Name, input.ValueFrom.HelmRef.Key)
	}
	if input.ValueFrom.ServiceAccount != nil {
		return GetServiceAccountTokenFromCache(ctx, namespace, *input.ValueFrom.ServiceAccount)
	}

	return "", nil
}

func GetEnvStringFromCache(ctx Context, env string, namespace string) (string, error) {
	var envvar types.EnvVar
	if err := envvar.Scan(env); err != nil {
		return "", err
	}
	return GetEnvValueFromCache(ctx, envvar, namespace)
}

func GetHelmValueFromCache(ctx Context, namespace, releaseName, key string) (string, error) {
	id := fmt.Sprintf("helm/%s/%s/%s", namespace, releaseName, key)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}

	keyJPExpr, err := jp.ParseString(key)
	if err != nil {
		return "", fmt.Errorf("could not parse key:%s. must be a valid jsonpath expression. %w", key, err)
	}

	secretList, err := ctx.Kubernetes().CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("type=%s", helmSecretType),
		LabelSelector: fmt.Sprintf("status=deployed,name=%s", releaseName),
		Limit:         1,
	})
	if err != nil {
		return "", fmt.Errorf("could not get secrets in namespace: %s: %w", namespace, err)
	}

	if len(secretList.Items) == 0 {
		return "", fmt.Errorf("a deployed helm secret was not found %s/%s", namespace, releaseName)
	}
	secret := secretList.Items[0]

	if secret.Name == "" {
		return "", fmt.Errorf("could not find helm secret %s/%s", namespace, releaseName)
	}

	release, err := base64.StdEncoding.DecodeString(string(secret.Data["release"]))
	if err != nil {
		return "", fmt.Errorf("could not base64 decode helm secret %s/%s: %w", namespace, secret.Name, err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(release))
	if err != nil {
		return "", fmt.Errorf("could not unzip helm secret %s/%s: %w", namespace, secret.Name, err)
	}

	var rawJson map[string]any
	if err := json.NewDecoder(gzipReader).Decode(&rawJson); err != nil {
		return "", fmt.Errorf("could not decode unzipped helm secret %s/%s: %w", namespace, secret.Name, err)
	}

	var chartValues any = map[string]any{}
	if chart, ok := rawJson["chart"].(map[string]any); ok {
		chartValues = chart["values"]
	}

	merged, info := jsonmerge.Merge(rawJson["config"], chartValues)
	if len(info.Errors) != 0 {
		return "", fmt.Errorf("could not merge helm config and values of helm secret %s/%s: %v", namespace, secret.Name, info.Errors)
	}

	results := keyJPExpr.Get(merged)
	if len(results) == 0 {
		return "", fmt.Errorf("could not find key %s in merged helm secret %s/%s: %w", key, namespace, secret.Name, err)
	}

	output, err := json.Marshal(results[0])
	if err != nil {
		return "", fmt.Errorf("could not marshal merged helm secret %s/%s: %w", namespace, secret.Name, err)
	}

	envCache.Set(id, string(output), 5*time.Minute)
	return string(output), nil
}

func GetSecretFromCache(ctx Context, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("secret/%s/%s/%s", namespace, name, key)

	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	secret, err := ctx.Kubernetes().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if secret == nil {
		return "", fmt.Errorf("could not get contents of secret %s/%s: %w", namespace, name, err)
	}

	value, ok := secret.Data[key]

	if !ok {
		names := []string{}
		for k := range secret.Data {
			names = append(names, k)
		}
		return "", fmt.Errorf("could not find key %v in secret %s/%s (%s)", key, namespace, name, strings.Join(names, ", "))
	}
	envCache.Set(id, string(value), 5*time.Minute)
	return string(value), nil
}

func GetConfigMapFromCache(ctx Context, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("cm/%s/%s/%s", namespace, name, key)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	configMap, err := ctx.Kubernetes().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if configMap == nil {
		return "", fmt.Errorf("could not get contents of configmap %s/%s: %w", namespace, name, err)
	}

	value, ok := configMap.Data[key]
	if !ok {
		names := []string{}
		for k := range configMap.Data {
			names = append(names, k)
		}
		return "", fmt.Errorf("could not find key %v in configmap %s/%s (%s)", key, namespace, name, strings.Join(names, ", "))
	}
	envCache.Set(id, string(value), 5*time.Minute)
	return string(value), nil
}

func GetServiceAccountTokenFromCache(ctx Context, namespace, serviceAccount string) (string, error) {
	id := fmt.Sprintf("sa-token/%s/%s", namespace, serviceAccount)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	tokenRequest, err := ctx.Kubernetes().CoreV1().ServiceAccounts(namespace).CreateToken(ctx, serviceAccount, &authenticationv1.TokenRequest{}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get token for service account %s/%s: %w", namespace, serviceAccount, err)
	}

	envCache.Set(id, tokenRequest.Status.Token, time.Until(tokenRequest.Status.ExpirationTimestamp.Time))
	return tokenRequest.Status.Token, nil
}
