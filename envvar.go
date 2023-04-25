package duty

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Create a cache with a default expiration time of 5 minutes, and which
// purges expired items every 10 minutes
var envCache = cache.New(5*time.Minute, 10*time.Minute)

func FindConnectionFromEnvVar(ctx context.Context, db *gorm.DB, input types.EnvVar) (*models.Connection, error) {
	if input.Connection == "" {
		return nil, nil
	}

	return FindConnectionByURL(ctx, db, input.Connection)
}

func FindConnectionFromEnvVarSource(ctx context.Context, db *gorm.DB, input types.EnvVarSource) (*models.Connection, error) {
	if input.Connection == "" {
		return nil, nil
	}

	return FindConnectionByURL(ctx, db, input.Connection)
}

func GetEnvValueFromCache(c kubernetes.Interface, input types.EnvVar, namespace string) (string, error) {
	if input.ValueFrom == nil {
		return input.ValueStatic, nil
	}
	if input.ValueFrom.SecretKeyRef != nil {
		value, err := GetSecretFromCache(c, namespace, input.ValueFrom.SecretKeyRef.Name, input.ValueFrom.SecretKeyRef.Key)
		return value, err
	}
	if input.ValueFrom.ConfigMapKeyRef != nil {
		value, err := GetConfigMapFromCache(c, namespace, input.ValueFrom.ConfigMapKeyRef.Name, input.ValueFrom.ConfigMapKeyRef.Key)
		return value, err
	}

	return "", nil
}

func GetEnvStringFromCache(c kubernetes.Interface, env string, namespace string) (string, error) {
	var envvar types.EnvVar
	if err := envvar.Scan(env); err != nil {
		return "", err
	}
	return GetEnvValueFromCache(c, envvar, namespace)
}

func GetSecretFromCache(c kubernetes.Interface, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("secret/%s/%s/%s", namespace, name, key)

	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	secret, err := c.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if secret == nil {
		return "", fmt.Errorf("Could not get contents of secret %v from namespace %v: %v", name, namespace, err)
	}

	value, ok := secret.Data[key]

	if !ok {
		names := []string{}
		for k := range secret.Data {
			names = append(names, k)
		}
		return "", fmt.Errorf("Could not find key %v in secret %v (%s)", key, name, strings.Join(names, ", "))
	}
	envCache.Set(id, string(value), 5*time.Minute)
	return string(value), nil
}

func GetConfigMapFromCache(c kubernetes.Interface, namespace, name, key string) (string, error) {
	id := fmt.Sprintf("cm/%s/%s/%s", namespace, name, key)
	if value, found := envCache.Get(id); found {
		return value.(string), nil
	}
	configMap, err := c.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if configMap == nil {
		return "", fmt.Errorf("Could not get contents of configmap %v from namespace %v: %v", name, namespace, err)
	}

	value, ok := configMap.Data[key]
	if !ok {
		names := []string{}
		for k := range configMap.Data {
			names = append(names, k)
		}
		return "", fmt.Errorf("Could not find key %v in configmap %v (%s)", key, name, strings.Join(names, ", "))
	}
	envCache.Set(id, string(value), 5*time.Minute)
	return string(value), nil
}
