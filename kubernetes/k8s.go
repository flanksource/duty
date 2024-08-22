package kubernetes

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/cache"
	"github.com/henvic/httpretty"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Nil = fake.NewSimpleClientset()

var sensitiveUrls = []*regexp.Regexp{
	regexp.MustCompile("/api/v1/namespaces/.*/secrets"),
	regexp.MustCompile("/api/v1/namespaces/.*/connections"),
	regexp.MustCompile("/api/v1/namespaces/.*/serviceaccounts/default/token"),
}

var kubeCache = cache.NewCache[kubeCacheData]("kube-clients", time.Hour)

type kubeCacheData struct {
	Client kubernetes.Interface
	Config *rest.Config
}

func cacheResult(key string, k kubernetes.Interface, c *rest.Config, e error) (kubernetes.Interface, *rest.Config, error) {
	if e != nil {
		return nil, nil, e
	}

	data := kubeCacheData{
		Client: k,
		Config: c,
	}

	_ = kubeCache.Set(context.TODO(), key, data)
	return k, c, e
}

func NewClient(logger logger.Logger, kubeconfigPaths ...string) (kubernetes.Interface, *rest.Config, error) {
	if len(kubeconfigPaths) == 0 {
		kubeconfigPaths = []string{os.Getenv("KUBECONFIG"), os.ExpandEnv("$HOME/.kube/config")}
	}

	for _, path := range kubeconfigPaths {
		if cached, _ := kubeCache.Get(context.TODO(), path); cached.Config != nil {
			return cached.Client, cached.Config, nil
		}
		if files.Exists(path) {
			if configBytes, err := os.ReadFile(path); err != nil {
				return nil, nil, err
			} else {
				logger.Infof("Using kubeconfig %s", path)
				client, config, err := NewClientWithConfig(logger, configBytes)
				return cacheResult(path, client, config, err)
			}
		}
	}

	inCluster := "in-cluster"
	if cached, _ := kubeCache.Get(context.TODO(), inCluster); cached.Config != nil {
		return cached.Client, cached.Config, nil
	}

	if config, err := rest.InClusterConfig(); err == nil {
		client, err := kubernetes.NewForConfig(trace(logger, config))
		return cacheResult(inCluster, client, config, err)
	}
	return Nil, nil, nil
}

func NewClientWithConfig(logger logger.Logger, kubeConfig []byte) (kubernetes.Interface, *rest.Config, error) {

	if cached, _ := kubeCache.Get(context.TODO(), string(kubeConfig)); cached.Config != nil {
		return cached.Client, cached.Config, nil
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	if config, err := clientConfig.ClientConfig(); err != nil {
		return nil, nil, err
	} else {
		client, err := kubernetes.NewForConfig(trace(logger, config))
		return cacheResult(string(kubeConfig), client, config, err)
	}
}

func trace(clogger logger.Logger, config *rest.Config) *rest.Config {
	if clogger.IsLevelEnabled(7) {
		clogger.Infof("tracing kubernetes API calls")
		logger := &httpretty.Logger{

			Time:           true,
			TLS:            clogger.IsLevelEnabled(8),
			RequestHeader:  true,
			RequestBody:    clogger.IsLevelEnabled(9),
			ResponseHeader: true,
			ResponseBody:   clogger.IsLevelEnabled(8),
			Colors:         true, // erase line if you don't like colors
			Formatters:     []httpretty.Formatter{&httpretty.JSONFormatter{}},
		}

		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logger.RoundTripper(rt)
		}
		logger.SetFilter(func(r *http.Request) (bool, error) {
			for _, url := range sensitiveUrls {
				if url.MatchString(r.URL.Path) {
					clogger.Tracef("%s %s (Skipping sensitive URL)", console.Greenf(r.Method), r.URL.Path)
					return true, nil
				}
			}
			return false, nil
		})
	}
	return config
}

// ExecutePodf runs the specified shell command inside a container of the specified pod
func GetClusterName(config *rest.Config) string {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return ""
	}
	kubeadmConfig, err := clientset.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return ""
	}
	clusterConfiguration := make(map[string]interface{})

	if err := yaml.Unmarshal([]byte(kubeadmConfig.Data["ClusterConfiguration"]), &clusterConfiguration); err != nil {
		return ""
	}
	return clusterConfiguration["clusterName"].(string)
}
