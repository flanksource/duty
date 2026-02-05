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
	"github.com/samber/lo"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/types"
	"github.com/patrickmn/go-cache"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create a cache with a default expiration time of 5 minutes, and which
// purges expired items every 10 minutes
var envCache = cache.New(5*time.Minute, 10*time.Minute)

const helmSecretType = "helm.sh/release.v1"

func GetEnvValueFromCache(ctx Context, input types.EnvVar, namespace string) (value string, err error) {
	if input.IsEmpty() {
		return "", nil
	}
	ctx, cancel := ctx.WithTimeout(ctx.Properties().Duration("envvar.lookup.timeout", 5*time.Second))
	defer cancel()
	if namespace == "" {
		namespace = ctx.GetNamespace()
	}
	var source = ""

	if input.ValueFrom == nil {
		source = "static"
		value = input.ValueStatic
	} else if input.ValueFrom.SecretKeyRef != nil && !input.ValueFrom.SecretKeyRef.IsEmpty() {
		source = fmt.Sprintf("secret(%s/%s).%s", namespace, input.ValueFrom.SecretKeyRef.Name, input.ValueFrom.SecretKeyRef.Key)
		value, err = GetSecretFromCache(ctx, namespace, input.ValueFrom.SecretKeyRef.Name, input.ValueFrom.SecretKeyRef.Key)
	} else if input.ValueFrom.ConfigMapKeyRef != nil && !input.ValueFrom.ConfigMapKeyRef.IsEmpty() {
		source = fmt.Sprintf("configmap(%s/%s).%s", namespace, input.ValueFrom.ConfigMapKeyRef.Name, input.ValueFrom.ConfigMapKeyRef.Key)
		value, err = GetConfigMapFromCache(ctx, namespace, input.ValueFrom.ConfigMapKeyRef.Name, input.ValueFrom.ConfigMapKeyRef.Key)
	} else if input.ValueFrom.HelmRef != nil && !input.ValueFrom.HelmRef.IsEmpty() {
		source = fmt.Sprintf("helm(%s/%s).%s", namespace, input.ValueFrom.HelmRef.Name, input.ValueFrom.HelmRef.Key)
		value, err = GetHelmValueFromCache(ctx, namespace, input.ValueFrom.HelmRef.Name, input.ValueFrom.HelmRef.Key)
	} else if !lo.IsEmpty(input.ValueFrom.ServiceAccount) {
		source = fmt.Sprintf("service-account(%s/%s)", namespace, *input.ValueFrom.ServiceAccount)
		value, err = GetServiceAccountTokenFromCache(ctx, namespace, *input.ValueFrom.ServiceAccount)
	}

	if err != nil {
		ctx.Logger.V(3).Infof("lookup[%s] failed %s => %s", input.Name, source, err.Error())
	} else if ctx.Logger.IsLevelEnabled(5) {
		ctx.Logger.V(5).Infof("lookup[%s] %s => %s", input.Name, source, logger.PrintableSecret(value))
	}

	return value, err
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

	client, err := ctx.LocalKubernetes()
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes client: %w", err)
	}

	secretList, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
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

	merged, info := jsonmerge.Merge(chartValues, rawJson["config"])
	if len(info.Errors) != 0 {
		return "", fmt.Errorf("could not merge helm config and values of helm secret %s/%s: %v", namespace, secret.Name, info.Errors)
	}

	results := keyJPExpr.Get(merged)
	if len(results) == 0 {
		return "", fmt.Errorf("could not find key %s in merged helm secret %s/%s: %w", key, namespace, secret.Name, err)
	}

	val := ""
	if len(results) == 1 {
		switch v := results[0].(type) {
		case string:
			val = v
		case []byte:
			val = string(v)
		case int, int32, int64:
			val = fmt.Sprintf("%d", v)
		case float32, float64:
			val = fmt.Sprintf("%0f", v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("could not marshal merged helm secret %s/%s: %w", namespace, secret.Name, err)
			}
			val = string(b)

		}
	}

	envCache.Set(id, val, ctx.Properties().Duration("envvar.helm.cache.timeout", ctx.Properties().Duration("envvar.cache.timeout", 5*time.Minute)))
	return val, nil
}

func GetSecretFromCache(ctx Context, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("secret/%s/%s/%s", namespace, name, key)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	client, err := ctx.LocalKubernetes()
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes client: %w", err)
	}

	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("could not find secret %s/%s: %s", namespace, name, err)
	}

	if secret == nil {
		return "", fmt.Errorf("could not get contents of secret %s/%s: %w", namespace, name, err)
	}

	value, ok := secret.Data[key]

	if !ok {
		return "", fmt.Errorf("could not find key %v in secret %s/%s (%s)", key, namespace, name, strings.Join(lo.Keys(secret.Data), ", "))
	}
	envCache.Set(id, string(value), ctx.Properties().Duration("envvar.cache.timeout", 5*time.Minute))
	return string(value), nil
}

func GetConfigMapFromCache(ctx Context, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("cm/%s/%s/%s", namespace, name, key)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	client, err := ctx.LocalKubernetes()
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes client: %w", err)
	}
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get configmap %s/%s: %s", namespace, name, err)
	}
	if configMap == nil {
		return "", fmt.Errorf("could not get contents of configmap %s/%s: %w", namespace, name, err)
	}

	value, ok := configMap.Data[key]
	if !ok {
		return "", fmt.Errorf("could not find key %v in configmap %s/%s (%s)", key, namespace, name,
			strings.Join(lo.Keys(configMap.Data), ", "))
	}
	envCache.Set(id, string(value), ctx.Properties().Duration("envvar.cache.timeout", 5*time.Minute))
	return string(value), nil
}

func GetServiceAccountTokenFromCache(ctx Context, namespace, serviceAccount string) (string, error) {
	id := fmt.Sprintf("sa-token/%s/%s", namespace, serviceAccount)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	client, err := ctx.LocalKubernetes()
	if err != nil {
		return "", fmt.Errorf("error creating kubernetes client: %w", err)
	}
	tokenRequest, err := client.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, serviceAccount, &authenticationv1.TokenRequest{}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get token for service account %s/%s: %w", namespace, serviceAccount, err)
	}

	envCache.Set(id, tokenRequest.Status.Token, time.Until(tokenRequest.Status.ExpirationTimestamp.Time))
	return tokenRequest.Status.Token, nil
}
