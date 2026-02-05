package kubernetes

import (
	"bufio"
	"bytes"
	"container/list"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	cachev4 "github.com/eko/gocache/lib/v4/cache"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
	"github.com/flanksource/commons/timer"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
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

	"github.com/flanksource/duty/cache"
	"github.com/flanksource/duty/types"
)

type gvkClientResourceCacheValue struct {
	gvr     schema.GroupVersionResource
	mapping *meta.RESTMapping
}

type Client struct {
	kubernetes.Interface
	restMapper             *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient          *dynamic.DynamicClient
	Config                 *rest.Config // Prefer updaating token in place
	gvkClientResourceCache cachev4.CacheInterface[gvkClientResourceCacheValue]
	logger                 logger.Logger
}

func (c *Client) SetLogger(logger logger.Logger) {
	c.logger = logger
}

func NewKubeClient(logger logger.Logger, client kubernetes.Interface, config *rest.Config) *Client {
	return &Client{
		Interface:              client,
		Config:                 config,
		gvkClientResourceCache: cache.NewCache[gvkClientResourceCacheValue]("gvk-cache", 24*time.Hour),
		logger:                 logger,
	}
}

func (c *Client) Reset() {
	c.dynamicClient = nil
}

func (c *Client) ResetRestMapper() {
	c.restMapper.Reset()
}

func (c *Client) FetchResources(
	ctx context.Context,
	resources ...unstructured.Unstructured,
) ([]unstructured.Unstructured, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	items := make(chan unstructured.Unstructured, len(resources))
	for i := range resources {
		resource := resources[i]
		client, err := c.GetClientByGroupVersionKind(
			ctx,
			resource.GroupVersionKind().Group,
			resource.GroupVersionKind().Version,
			resource.GetKind(),
		)
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

func (c *Client) GetClientByGroupVersionKind(
	ctx context.Context, group, version, kind string,
) (dynamic.NamespaceableResourceInterface, error) {
	dynamicClient, err := c.GetDynamicClient()
	if err != nil {
		return nil, err
	}

	cacheKey := group + version + kind
	if res, err := c.gvkClientResourceCache.Get(ctx, cacheKey); err == nil {
		return dynamicClient.Resource(res.gvr), nil
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

	_ = c.gvkClientResourceCache.Set(ctx, cacheKey, gvkClientResourceCacheValue{gvr: mapping.Resource, mapping: mapping})
	return dynamicClient.Resource(mapping.Resource), nil
}

func (c *Client) RestConfig() *rest.Config {
	return c.Config
}

// WARN: "Kind" is not specific enough.
// A cluster can have various resources with the same Kind.
// example: helmchrats.helm.cattle.io & helmcharts.source.toolkit.fluxcd.io both have HelmChart as the kind.
//
// Use GetClientByGroupVersionKind instead.
func (c *Client) GetClientByKind(kind string) (dynamic.NamespaceableResourceInterface, *meta.RESTMapping, error) {
	dynamicClient, err := c.GetDynamicClient()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dynamic client: %w", err)
	}

	if res, err := c.gvkClientResourceCache.Get(context.Background(), kind); err == nil {
		return dynamicClient.Resource(res.gvr), res.mapping, nil
	}

	rm, _ := c.GetRestMapper()
	gvk, err := rm.KindFor(schema.GroupVersionResource{
		Resource: kind,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get kind for %s: %w", kind, err)
	}

	gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
	mapping, err := rm.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rest mapping for %s: %w", kind, err)
	}

	if err := c.gvkClientResourceCache.Set(context.Background(), kind, gvkClientResourceCacheValue{gvr: mapping.Resource, mapping: mapping}); err != nil {
		c.logger.Errorf("failed to set gvk cache for %s: %s", kind, err)
	}

	return dynamicClient.Resource(mapping.Resource), mapping, nil
}

// ParseAPIVersionKind parses an apiVersion/Kind string into a GroupVersionKind.
// This format allows specifying the exact API group, version, and kind to avoid
// ambiguity when multiple resources share the same Kind name.
//
// Supported formats:
//   - "version/kind" for core API group (e.g., "v1/Pod", "v1/Service")
//   - "group/version/kind" for named API groups (e.g., "apps/v1/Deployment")
//   - "domain.group/version/kind" for domain-based groups (e.g., "serving.knative.dev/v1/Service")
//
// Examples:
//   - "v1/Pod" → {Group: "", Version: "v1", Kind: "Pod"}
//   - "apps/v1/Deployment" → {Group: "apps", Version: "v1", Kind: "Deployment"}
//   - "serving.knative.dev/v1/Service" → {Group: "serving.knative.dev", Version: "v1", Kind: "Service"}
//
// This is useful when you need to distinguish between resources with the same Kind
// but different API groups (e.g., v1/Service vs serving.knative.dev/v1/Service).
func ParseAPIVersionKind(apiVersionKind string) (schema.GroupVersionKind, error) {
	parts := strings.Split(apiVersionKind, "/")

	switch len(parts) {
	case 2:
		// Format: version/kind (e.g., "v1/Pod" for core API group)
		return schema.GroupVersionKind{
			Group:   "",
			Version: parts[0],
			Kind:    parts[1],
		}, nil
	case 3:
		// Format: group/version/kind (e.g., "apps/v1/Deployment", "serving.knative.dev/v1/Service")
		return schema.GroupVersionKind{
			Group:   parts[0],
			Version: parts[1],
			Kind:    parts[2],
		}, nil
	default:
		return schema.GroupVersionKind{}, fmt.Errorf("invalid apiVersion/Kind format: %q (expected \"version/Kind\" or \"group/version/Kind\")", apiVersionKind)
	}
}

func (c *Client) DeleteByGVK(ctx context.Context, namespace, name string, gvk schema.GroupVersionKind) (bool, error) {
	client, err := c.GetClientByGroupVersionKind(ctx, gvk.Group, gvk.Version, gvk.Kind)
	if err != nil {
		return false, err
	}

	if err := client.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
	}

	return true, nil
}

// GetDynamicClient creates a new k8s client
func (c *Client) GetDynamicClient() (dynamic.Interface, error) {
	if c.dynamicClient != nil {
		return c.dynamicClient, nil
	}

	c.logger.V(3).Infof("creating new dynamic client")
	var err error
	c.dynamicClient, err = dynamic.NewForConfig(c.Config)
	return c.dynamicClient, err
}

func (c *Client) GetRestMapper() (meta.RESTMapper, error) {
	if c.restMapper != nil {
		return c.restMapper, nil
	}

	// re-use kubectl cache
	host := c.Config.Host
	host = strings.ReplaceAll(host, "https://", "")
	host = strings.ReplaceAll(host, "-", "_")
	host = strings.ReplaceAll(host, ":", "_")
	cacheDir := os.ExpandEnv("$HOME/.kube/cache/discovery/" + host)
	timeout := properties.Duration(240*time.Minute, "kubernetes.cache.timeout")
	c.logger.V(3).Infof("creating new rest mapper with cache dir: %s and timeout: %s", cacheDir, timeout)
	cache, err := disk.NewCachedDiscoveryClientForConfig(
		c.Config,
		cacheDir,
		"",
		timeout,
	)
	if err != nil {
		return nil, err
	}
	c.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cache)
	return c.restMapper, err
}

func (c *Client) ExecutePodf(
	ctx context.Context,
	namespace, pod, container string,
	command ...string,
) (string, string, error) {
	const tty = false
	req := c.CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(c.Config, "POST", req.URL())
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

func (c *Client) GetPodLogs(ctx context.Context, namespace, podName, container string) (io.ReadCloser, error) {
	podLogOptions := v1.PodLogOptions{}
	if container != "" {
		podLogOptions.Container = container
	}

	req := c.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}

	return podLogs, nil
}

// WaitForPod waits for a pod to be in the specified phase, or returns an
// error if the timeout is exceeded
func (c *Client) WaitForPod(
	ctx context.Context,
	namespace, name string,
	timeout time.Duration,
	phases ...v1.PodPhase,
) error {
	start := time.Now()

	pods := c.CoreV1().Pods(namespace)
	for {
		pod, err := pods.Get(ctx, name, metav1.GetOptions{})
		if start.Add(timeout).Before(time.Now()) {
			return fmt.Errorf("timeout exceeded waiting for %s is %s, error: %v", name, pod.Status.Phase, err)
		}

		if pod == nil || pod.Status.Phase == v1.PodPending {
			time.Sleep(5 * time.Second)
			continue
		}
		if pod.Status.Phase == v1.PodFailed {
			return nil
		}

		for _, phase := range phases {
			if pod.Status.Phase == phase {
				return nil
			}
		}
	}
}

func (c *Client) StreamLogsV2(
	ctx context.Context,
	namespace, name string,
	timeout time.Duration,
	containerNames ...string,
) error {
	podsClient := c.CoreV1().Pods(namespace)
	pod, err := podsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if err := c.WaitForContainerStart(ctx, namespace, name, 120*time.Second, containerNames...); err != nil {
		return err
	}

	var wg sync.WaitGroup
	containers := list.New()

	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if len(containerNames) == 0 || lo.Contains(containerNames, container.Name) {
			containers.PushBack(container)
		}
	}

	// Loop over container list.
	for element := containers.Front(); element != nil; element = element.Next() {
		container := element.Value.(v1.Container)
		logs := podsClient.GetLogs(pod.Name, &v1.PodLogOptions{
			Container: container.Name,
		})

		prefix := pod.Name
		if len(pod.Spec.Containers) > 1 {
			prefix += "/" + container.Name
		}

		podLogs, err := logs.Stream(ctx)
		if err != nil {
			containers.PushBack(container)
			logger.Tracef("failed to begin streaming %s/%s: %s", pod.Name, container.Name, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		wg.Add(1)

		go func() {
			defer podLogs.Close()
			defer wg.Done()

			scanner := bufio.NewScanner(podLogs)
			for scanner.Scan() {
				incoming := scanner.Bytes()
				buffer := make([]byte, len(incoming))
				copy(buffer, incoming)
				fmt.Printf("\x1b[38;5;244m[%s]\x1b[0m %s\n", prefix, string(buffer))
			}
		}()
	}

	wg.Wait()

	if err = c.WaitForPod(ctx, namespace, name, timeout, v1.PodSucceeded); err != nil {
		return err
	}

	pod, err = podsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod.Status.Phase == v1.PodSucceeded {
		return nil
	}

	return fmt.Errorf("pod did not finish successfully %s - %s", pod.Status.Phase, pod.Status.Message)
}

// WaitForContainerStart waits for the specified containers to be started (or any container if no names are specified) - returns an error if the timeout is exceeded
func (c *Client) WaitForContainerStart(
	ctx context.Context,
	namespace, name string,
	timeout time.Duration,
	containerNames ...string,
) error {
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	podsClient := c.CoreV1().Pods(namespace)
	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout exceeded waiting for %s", name)

		case <-ctx.Done():
			return ctx.Err()

		default:
			pod, err := podsClient.Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if apiErrors.IsNotFound(err) {
					time.Sleep(time.Second)
					continue
				}

				return err
			}

			for _, container := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
				if len(containerNames) > 0 && !lo.Contains(containerNames, container.Name) {
					continue
				}

				if container.State.Running != nil || container.State.Terminated != nil {
					return nil
				}
			}

			time.Sleep(time.Second)
		}
	}
}

func (c *Client) ExpandNamespaces(ctx context.Context, namespace string) ([]string, error) {
	namespaces := []string{}
	if namespace == "*" || namespace == "" {
		return []string{""}, nil
	}

	if !types.IsMatchItem(namespace) {
		return []string{namespace}, nil
	}

	resources, err := c.QueryResources(ctx, types.ResourceSelector{Name: namespace}.Type("Namesapce"))
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		namespaces = append(namespaces, resource.GetName())
	}

	return namespaces, nil
}

func (c *Client) QueryResources(ctx context.Context, selector types.ResourceSelector) ([]unstructured.Unstructured, error) {
	timer := timer.NewTimer()

	var resources []unstructured.Unstructured
	for _, apiVersionKind := range selector.Types {
		if strings.ToLower(apiVersionKind) == "namespace" && selector.IsMetadataOnly() {
			if name, ok := selector.ToGetOptions(); ok {
				return []unstructured.Unstructured{{
					Object: map[string]any{
						"apiVersion:": "v1",
						"kind":        "Namespace",
						"metadata": map[string]any{
							"name": name,
						},
					},
				}}, nil
			}
		}

		apiVersionKind = strings.TrimPrefix(apiVersionKind, "Kubernetes::")

		var client dynamic.NamespaceableResourceInterface
		var rm *meta.RESTMapping
		var err error

		// Check if kind uses apiVersion/Kind format (e.g., "v1/Pod", "apps/v1/Deployment")
		if strings.Contains(apiVersionKind, "/") {
			gvk, err := ParseAPIVersionKind(apiVersionKind)
			if err != nil {
				return nil, err
			}

			client, err = c.GetClientByGroupVersionKind(ctx, gvk.Group, gvk.Version, gvk.Kind)
			if err != nil {
				return nil, fmt.Errorf("failed to get client for %s: %w", apiVersionKind, err)
			}

			restMapper, _ := c.GetRestMapper()
			rm, err = restMapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
			if err != nil {
				return nil, fmt.Errorf("failed to get rest mapping for %s: %w", apiVersionKind, err)
			}
		} else {
			client, rm, err = c.GetClientByKind(apiVersionKind)
			if err != nil {
				return nil, fmt.Errorf("failed to get client for %s: %w", apiVersionKind, err)
			}
		}

		isClusterScoped := rm.Scope.Name() == meta.RESTScopeNameRoot

		var namespaces []string
		if isClusterScoped {
			namespaces = []string{""}
		} else {
			namespaces, err = c.ExpandNamespaces(ctx, selector.Namespace)
			if apiErrors.IsNotFound(err) {
				continue
			} else if err != nil {
				return nil, fmt.Errorf("failed to expand namespaces for %s: %w", apiVersionKind, err)
			}
		}

		for _, namespace := range namespaces {
			cc := client.Namespace(namespace)
			if isClusterScoped {
				cc = client
			}

			if name, ok := selector.ToGetOptions(); ok && !types.IsMatchItem(name) {
				resource, err := cc.Get(ctx, name, metav1.GetOptions{})
				if apiErrors.IsNotFound(err) {
					continue
				} else if err != nil {
					return nil, fmt.Errorf("failed to get resource %s/%s: %w", namespace, name, err)
				}

				resources = append(resources, *resource)
				continue
			}

			list, full := selector.ToListOptions()
			resourceList, err := cc.List(ctx, list)
			if err != nil {
				return nil, fmt.Errorf("failed to list resources %s/%s: %w", namespace, selector.Name, err)
			}

			if full {
				resources = append(resources, resourceList.Items...)
				continue
			}

			for _, resource := range resourceList.Items {
				if selector.Matches(&types.UnstructuredResource{Unstructured: &resource}) {
					resources = append(resources, resource)
				}
			}
		}
	}

	c.logger.Debugf("%s => count=%d duration=%s", selector, len(resources), timer)
	return resources, nil
}

func safeString(buf *bytes.Buffer) string {
	if buf == nil || buf.Len() == 0 {
		return ""
	}
	return buf.String()
}
