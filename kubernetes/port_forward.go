package kubernetes

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwardOptions configures port forwarding to a Kubernetes resource.
// Either Name or LabelSelector must be provided to identify the target resource.
type PortForwardOptions struct {
	Namespace     string `json:"namespace,omitempty"`
	Name          string `json:"name,omitempty"`
	LabelSelector string `json:"labelSelector,omitempty"`
	RemotePort    int    `json:"remotePort,omitempty"`
	Kind          string `json:"kind"`
}

func PortForward(ctx context.Context, k8s kubernetes.Interface, restConfig *rest.Config, opts PortForwardOptions) (int, chan struct{}, error) {
	if err := opts.validate(); err != nil {
		return 0, nil, err
	}
	switch opts.Kind {
	case "pod":
		return PortForwardPod(ctx, k8s, restConfig, opts)
	case "deployment":
		return PortForwardDeployment(ctx, k8s, restConfig, opts)
	case "service":
		return PortForwardService(ctx, k8s, restConfig, opts)
	}

	// This never happens since type is validated in opts.validate()
	return 0, nil, fmt.Errorf("invalid kind:%s", opts.Kind)
}

// PortForwardPod sets up port forwarding to a pod matching the given label selector.
// Returns the local port, a stop channel to close when done, and any error.
func PortForwardPod(ctx context.Context, k8s kubernetes.Interface, restConfig *rest.Config, opts PortForwardOptions) (int, chan struct{}, error) {
	if err := opts.validate(); err != nil {
		return 0, nil, err
	}

	var pod *corev1.Pod
	if opts.Name != "" {
		p, err := k8s.CoreV1().Pods(opts.Namespace).Get(ctx, opts.Name, metav1.GetOptions{})
		if err != nil {
			return 0, nil, fmt.Errorf("pod %s not found: %w", opts.Name, err)
		}
		pod = p
	} else {
		pods, err := k8s.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: opts.LabelSelector,
		})
		if err != nil {
			return 0, nil, fmt.Errorf("failed to list pods: %w", err)
		}
		if len(pods.Items) == 0 {
			return 0, nil, fmt.Errorf("no pods found matching selector %s", opts.LabelSelector)
		}
		pod = &pods.Items[0]
	}

	remotePort, err := getRemotePort(opts.RemotePort, pod)
	if err != nil {
		return 0, nil, err
	}

	return portForwardToPod(ctx, restConfig, opts.Namespace, pod.Name, remotePort)
}

// PortForwardService sets up port forwarding to a pod backing the specified service.
// The service is found by Name or LabelSelector. Returns the local port, a stop channel, and any error.
func PortForwardService(ctx context.Context, k8s kubernetes.Interface, restConfig *rest.Config, opts PortForwardOptions) (int, chan struct{}, error) {
	if err := opts.validate(); err != nil {
		return 0, nil, err
	}

	var svcSelector map[string]string

	if opts.Name != "" {
		svc, err := k8s.CoreV1().Services(opts.Namespace).Get(ctx, opts.Name, metav1.GetOptions{})
		if err != nil {
			return 0, nil, fmt.Errorf("service %s not found: %w", opts.Name, err)
		}
		svcSelector = svc.Spec.Selector
	} else {
		services, err := k8s.CoreV1().Services(opts.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: opts.LabelSelector,
		})
		if err != nil {
			return 0, nil, fmt.Errorf("failed to list services: %w", err)
		}
		if len(services.Items) == 0 {
			return 0, nil, fmt.Errorf("no services found matching selector %s", opts.LabelSelector)
		}
		svcSelector = services.Items[0].Spec.Selector
	}

	if len(svcSelector) == 0 {
		return 0, nil, fmt.Errorf("service has no selector")
	}

	pods, err := k8s.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(svcSelector).String(),
	})
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list pods for service: %w", err)
	}
	if len(pods.Items) == 0 {
		return 0, nil, fmt.Errorf("no pods found for service")
	}

	remotePort, err := getRemotePort(opts.RemotePort, &pods.Items[0])
	if err != nil {
		return 0, nil, err
	}

	return portForwardToPod(ctx, restConfig, opts.Namespace, pods.Items[0].Name, remotePort)
}

// PortForwardDeployment sets up port forwarding to a pod managed by the specified deployment.
// The deployment is found by Name or LabelSelector. Returns the local port, a stop channel, and any error.
func PortForwardDeployment(ctx context.Context, k8s kubernetes.Interface, restConfig *rest.Config, opts PortForwardOptions) (int, chan struct{}, error) {
	if err := opts.validate(); err != nil {
		return 0, nil, err
	}

	var podSelector map[string]string

	if opts.Name != "" {
		deployment, err := k8s.AppsV1().Deployments(opts.Namespace).Get(ctx, opts.Name, metav1.GetOptions{})
		if err != nil {
			return 0, nil, fmt.Errorf("deployment %s not found: %w", opts.Name, err)
		}
		podSelector = deployment.Spec.Selector.MatchLabels
	} else {
		deployments, err := k8s.AppsV1().Deployments(opts.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: opts.LabelSelector,
		})
		if err != nil {
			return 0, nil, fmt.Errorf("failed to list deployments: %w", err)
		}
		if len(deployments.Items) == 0 {
			return 0, nil, fmt.Errorf("no deployments found matching selector %s", opts.LabelSelector)
		}
		podSelector = deployments.Items[0].Spec.Selector.MatchLabels
	}

	if len(podSelector) == 0 {
		return 0, nil, fmt.Errorf("deployment has no pod selector")
	}

	pods, err := k8s.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(podSelector).String(),
	})
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list pods for deployment: %w", err)
	}
	if len(pods.Items) == 0 {
		return 0, nil, fmt.Errorf("no pods found for deployment")
	}

	remotePort, err := getRemotePort(opts.RemotePort, &pods.Items[0])
	if err != nil {
		return 0, nil, err
	}

	return portForwardToPod(ctx, restConfig, opts.Namespace, pods.Items[0].Name, remotePort)
}

func (o PortForwardOptions) validate() error {
	if !slices.Contains([]string{"pod", "service", "deployment"}, o.Kind) {
		return fmt.Errorf("type[%s] should be one of pod, service, deployment", o.Kind)
	}
	if o.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if o.Name == "" && o.LabelSelector == "" {
		return fmt.Errorf("either Name or LabelSelector must be provided")
	}
	return nil
}

// getRemotePort returns the port to forward to. If remotePort is specified (> 0),
// it returns that. Otherwise, it returns the first container port from the pod.
func getRemotePort(remotePort int, pod *corev1.Pod) (int, error) {
	if remotePort > 0 {
		return remotePort, nil
	}

	for _, container := range pod.Spec.Containers {
		if len(container.Ports) > 0 {
			return int(container.Ports[0].ContainerPort), nil
		}
	}

	return 0, fmt.Errorf("pod %s has no container ports and remotePort was not specified", pod.Name)
}

// portForwardToPod establishes port forwarding to a specific pod.
func portForwardToPod(ctx context.Context, restConfig *rest.Config, namespace, podName string, remotePort int) (int, chan struct{}, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get free port: %w", err)
	}
	localPort := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse server URL: %w", err)
	}
	serverURL.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, serverURL)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	pf, err := portforward.New(dialer, ports, stopChan, readyChan, nil, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- pf.ForwardPorts()
	}()

	select {
	case <-readyChan:
		return localPort, stopChan, nil
	case err := <-errChan:
		return 0, nil, fmt.Errorf("port forward failed: %w", err)
	case <-ctx.Done():
		close(stopChan)
		return 0, nil, ctx.Err()
	}
}
