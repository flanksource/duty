package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	netURL "net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/flanksource/commons/console"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/cache"
	"github.com/henvic/httpretty"
	"github.com/samber/lo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var Nil = fake.NewSimpleClientset()

var sensitiveUrls = []*regexp.Regexp{
	regexp.MustCompile("/api/v1/namespaces/.*/secrets"),
	regexp.MustCompile("/api/v1/namespaces/.*/connections"),
	regexp.MustCompile("/api/v1/namespaces/.*/serviceaccounts/default/token"),
}

func NewClient(log logger.Logger, kubeconfigPaths ...string) (kubernetes.Interface, *rest.Config, error) {
	if len(kubeconfigPaths) == 0 {
		kubeconfigPaths = []string{os.Getenv("KUBECONFIG"), os.ExpandEnv("$HOME/.kube/config")}
	}

	for _, path := range kubeconfigPaths {
		if files.Exists(path) {
			if configBytes, err := os.ReadFile(path); err != nil {
				return nil, nil, err
			} else {
				log.Infof("Using kubeconfig %s", path)
				return NewClientWithConfig(log, configBytes)
			}
		}
	}

	if config, err := rest.InClusterConfig(); err == nil {
		client, err := kubernetes.NewForConfig(trace(log, config))
		return client, config, err
	}
	return Nil, nil, nil
}

func NewClientWithConfig(log logger.Logger, kubeConfig []byte) (kubernetes.Interface, *rest.Config, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	apiConfig, err := clientConfig.MergedRawConfig()
	if err != nil {
		return nil, nil, err
	}
	name, server := GetClusterNameFromKubeconfig(&apiConfig)
	_ = clusterNames.Set(context.TODO(), server, name)

	if config, err := clientConfig.ClientConfig(); err != nil {
		return nil, nil, err
	} else {
		client, err := kubernetes.NewForConfig(trace(logger.GetLogger("k8s."+name), config))
		return client, config, err
	}
}

func NewClientFromPathOrConfig(
	logger logger.Logger,
	kubeconfigOrPath string,
) (kubernetes.Interface, *rest.Config, error) {
	var client kubernetes.Interface
	var rest *rest.Config
	var err error

	if _, pathErr := os.Stat(kubeconfigOrPath); pathErr == nil {
		if client, rest, err = NewClient(logger, kubeconfigOrPath); err != nil {
			return nil, nil, err
		}
	} else {
		if client, rest, err = NewClientWithConfig(logger, []byte(kubeconfigOrPath)); err != nil {
			return nil, nil, err
		}
	}

	return client, rest, err
}

var clusterNames = cache.NewCache[string]("clusterNames", time.Hour*24)

func trace(clogger logger.Logger, config *rest.Config) *rest.Config {
	clogger.Tracef("creating new client-go for %s ", GetClusterName(config))
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
					clogger.Tracef("%s %s (Skipping sensitive URL)", console.Greenf("%s", r.Method), r.URL.Path)
					return true, nil
				}
			}
			return false, nil
		})
	}
	return config
}

func argsToMap(args []string) map[string]string {
	m := make(map[string]string)
	k := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			k = arg[2:]
		} else if strings.HasPrefix(arg, "-") {
			k = arg[1:]
		} else if k != "" {
			m[k] = arg
			k = ""
		}

	}
	return m
}

func GetClusterNameFromKubeconfig(config *clientcmdapi.Config) (clusterName string, server string) {
	auth := config.AuthInfos[config.CurrentContext]
	cluster := config.Clusters[config.CurrentContext]
	if cluster != nil {
		server = cluster.Server
	}

	if auth != nil && auth.Exec != nil {
		if strings.Contains(auth.Exec.Command, "gcloud") {
			clusterName = "gke:" + config.CurrentContext
			return
		}
		if auth.Exec.Command == "aws" {
			args := argsToMap(auth.Exec.Args)
			if name, ok := args["cluster-name"]; ok {
				clusterName = "eks:" + name
				return
			}
		}
	}

	if !lo.Contains([]string{"", "default", "gke_default", "kubernetes"}, config.CurrentContext) {
		// context name is usually more descriptive
		clusterName = "kubeconfig:" + config.CurrentContext
	}
	return clusterName, server
}

// GetClusterName returns the name of the cluster
func GetClusterName(config *rest.Config) string {
	if name, err := clusterNames.Get(context.TODO(), config.Host); err == nil && name != "" {
		return name
	}

	clusterName := ""
	if config.ExecProvider != nil {
		switch config.ExecProvider.Command {
		case "aws":
			args := argsToMap(config.ExecProvider.Args)
			if name, ok := args["cluster-name"]; ok {
				clusterName = "eks:" + name
			}
		case "gke-gcloud-auth-plugin":
			args := argsToMap(config.ExecProvider.Args)
			if name, ok := args["cluster"]; ok {
				clusterName = "gke:" + name
			}
		}
	}

	if clusterName != "" {
		_ = clusterNames.Set(context.TODO(), config.Host, clusterName)
		return clusterName
	}

	return config.Host
}

func GetTransportRoundtripper(config *rest.Config) (func(http.RoundTripper) http.RoundTripper, error) {
	k8srt, err := rest.TransportFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get transport config for k8s: %w", err)
	}

	return func(rt http.RoundTripper) http.RoundTripper {
		return k8srt
	}, nil
}

func GetProxiedURL(url string, config *rest.Config) (string, error) {
	parsedURL, err := netURL.Parse(url)
	if err != nil {
		return "", fmt.Errorf("error parsing url[%s]: %w", url, err)
	}

	port := lo.CoalesceOrEmpty(parsedURL.Port(), lo.Ternary(parsedURL.Scheme == "https", "443", "80"))
	parts := strings.Split(parsedURL.Hostname(), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("host[%s] is invalid. Use `service.namespace` format", parsedURL.Hostname())
	}
	svc, ns := parts[0], parts[1]
	path := strings.TrimPrefix(parsedURL.EscapedPath(), "/")
	proxyURL := fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s:%s/proxy/%s", config.Host, ns, svc, port, path)
	if parsedURL.RawQuery != "" {
		proxyURL += "?" + parsedURL.RawQuery
	}
	return proxyURL, nil
}
