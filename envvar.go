package duty

import (
	gocontext "context"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
	"k8s.io/client-go/kubernetes"
)

// deprecated use the methods in the context package
func GetEnvValueFromCache(c kubernetes.Interface, input types.EnvVar, namespace string) (string, error) {
	return context.GetEnvValueFromCache(context.NewContext(gocontext.TODO()).WithKubernetes(c), input, namespace)
}

// deprecated use the methods in the context package
func GetEnvStringFromCache(c kubernetes.Interface, env string, namespace string) (string, error) {
	return context.GetEnvStringFromCache(context.NewContext(gocontext.TODO()).WithKubernetes(c), env, namespace)
}

// deprecated use the methods in the context package
func GetSecretFromCache(c kubernetes.Interface, namespace, name, key string) (string, error) {
	return context.GetSecretFromCache(context.NewContext(gocontext.TODO()).WithKubernetes(c), namespace, name, key)
}

// deprecated use the methods in the context package
func GetConfigMapFromCache(c kubernetes.Interface, namespace, name, key string) (string, error) {
	return context.GetConfigMapFromCache(context.NewContext(gocontext.TODO()).WithKubernetes(c), namespace, name, key)
}
