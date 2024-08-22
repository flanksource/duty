package kubernetes

import (
	"context"
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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// DynamicClient is an updated & stripped of kommons client
type DynamicClient struct {
	restMapper    *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient *dynamic.DynamicClient
	config        *rest.Config
}

func NewKubeClient(config *rest.Config) *DynamicClient {
	return &DynamicClient{config: config}
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
