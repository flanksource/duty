package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flanksource/commons/properties"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/remotecommand"
)

// DynamicClient is an updated & stripped of kommons client
type DynamicClient struct {
	client        kubernetes.Interface
	restMapper    *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient *dynamic.DynamicClient
	config        *rest.Config
}

func NewKubeClient(client kubernetes.Interface, config *rest.Config) *DynamicClient {
	return &DynamicClient{config: config, client: client}
}

func (c *DynamicClient) FetchResources(ctx context.Context, resources ...unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	var items = make(chan unstructured.Unstructured, len(resources))
	for i := range resources {
		resource := resources[i]
		client, err := c.GetClientByGroupVersionKind(resource.GroupVersionKind().Group, resource.GroupVersionKind().Version, resource.GetKind())
		if err != nil {
			return nil, err
		}

		eg.Go(func() error {
			item, err := client.Namespace(resource.GetNamespace()).Get(ctx, resource.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}

			items <- *item
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	output, _, _, _ := lo.Buffer(items, len(items)) //nolint:dogsled
	return output, nil
}

func (c *DynamicClient) GetClientByGroupVersionKind(group, version, kind string) (dynamic.NamespaceableResourceInterface, error) {
	dynamicClient, err := c.GetDynamicClient()
	if err != nil {
		return nil, err
	}

	rm, _ := c.GetRestMapper()
	gvk, err := rm.KindFor(schema.GroupVersionResource{
		Resource: kind,
		Group:    group,
		Version:  version,
	})
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}

	return dynamicClient.Resource(mapping.Resource), nil
}

func (c *DynamicClient) GetClientByKind(kind string) (dynamic.NamespaceableResourceInterface, error) {
	dynamicClient, err := c.GetDynamicClient()
	if err != nil {
		return nil, err
	}
	rm, _ := c.GetRestMapper()
	gvk, err := rm.KindFor(schema.GroupVersionResource{
		Resource: kind,
	})
	if err != nil {
		return nil, err
	}
	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(mapping.Resource), nil
}

// GetDynamicClient creates a new k8s client
func (c *DynamicClient) GetDynamicClient() (dynamic.Interface, error) {
	if c.dynamicClient != nil {
		return c.dynamicClient, nil
	}

	var err error
	c.dynamicClient, err = dynamic.NewForConfig(c.config)
	return c.dynamicClient, err
}

func (c *DynamicClient) GetRestMapper() (meta.RESTMapper, error) {
	if c.restMapper != nil {
		return c.restMapper, nil
	}

	// re-use kubectl cache
	host := c.config.Host
	host = strings.ReplaceAll(host, "https://", "")
	host = strings.ReplaceAll(host, "-", "_")
	host = strings.ReplaceAll(host, ":", "_")
	cacheDir := os.ExpandEnv("$HOME/.kube/cache/discovery/" + host)
	cache, err := disk.NewCachedDiscoveryClientForConfig(c.config, cacheDir, "", properties.Duration(10*time.Minute, "kubernetes.cache.timeout"))
	if err != nil {
		return nil, err
	}
	c.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cache)
	return c.restMapper, err
}

func (c *DynamicClient) ExecutePodf(ctx context.Context, namespace, pod, container string, command ...string) (string, string, error) {
	const tty = false
	req := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container).
		Param("stdin", fmt.Sprintf("%t", false)).
		Param("stdout", fmt.Sprintf("%t", true)).
		Param("stderr", fmt.Sprintf("%t", true)).
		Param("tty", fmt.Sprintf("%t", tty))

	for _, c := range command {
		req.Param("command", c)
	}

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("ExecutePodf: Failed to get SPDY Executor: %v", err)
	}
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    tty,
	})

	_stdout := safeString(&stdout)
	_stderr := safeString(&stderr)

	if err != nil {
		return "", "", fmt.Errorf("failed to execute command: %v, stdout=%s stderr=%s", err, _stdout, _stderr)
	}

	return _stdout, _stderr, nil
}

func safeString(buf *bytes.Buffer) string {
	if buf == nil || buf.Len() == 0 {
		return ""
	}
	return buf.String()
}
