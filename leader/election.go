package leader

import (
	gocontext "context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/shutdown"
)

var (
	hostname     string
	podNamespace string
	service      string
)

const namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func getPodNamespace() (string, error) {
	// passed using K8s downwards API
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns, nil
	}

	data, err := os.ReadFile(namespaceFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %w", err)
	}

	ns := strings.TrimSpace(string(data))
	if ns == "" {
		return "", errors.New("namespace was neither found in the env nor in the service account path")
	}

	return ns, nil
}

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// To test locally
	if v, ok := os.LookupEnv("MC_HOSTNAME_OVERRIDE"); ok {
		logger.Infof("hostname overriden by MC_HOSTNAME_OVERRIDE: %s", v)
		hostname = v
	}

	if n, err := getPodNamespace(); err == nil {
		podNamespace = n
	}

	// Not sure if this is a very reliable way to get the service name
	service = strings.Split(hostname, "-")[0]
}

func Register(
	ctx context.Context,
	namespace string,
	onLead func(ctx gocontext.Context),
	onStoppedLead func(),
	onNewLeader func(identity string),
) error {
	if namespace == "" {
		namespace = podNamespace
	}

	ctx = ctx.WithNamespace(namespace)

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      service,
			Namespace: namespace,
		},
		Client: ctx.Kubernetes().CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: hostname,
		},
	}

	electionConfig := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   ctx.Properties().Duration("leader.lease.duration", 30*time.Second),
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leadCtx gocontext.Context) {
				updateLeaderLabel(ctx)
				onLead(leadCtx)
			},
			OnStoppedLeading: onStoppedLead,
			OnNewLeader: func(identity string) {
				if identity == hostname {
					return
				}

				onNewLeader(identity)
			},
		},
	}

	elector, err := leaderelection.NewLeaderElector(electionConfig)
	if err != nil {
		return err
	}

	leaderContext, cancel := gocontext.WithCancel(ctx)
	shutdown.AddHook(func() {
		cancel()

		// give the elector some time to release the lease
		time.Sleep(time.Second * 2)
	})

	go elector.Run(leaderContext)
	<-ctx.Done()

	return nil
}

// updateLeaderLabel sets leader:true label on the current pod
// and also removes that label from all other replicas.
func updateLeaderLabel(ctx context.Context) {
	backoff := retry.WithMaxRetries(3, retry.NewExponential(time.Second))
	err := retry.Do(ctx, backoff, func(_ctx gocontext.Context) error {
		pods, err := getAllReplicas(ctx, hostname)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to get replicas: %w", err))
		}

		for _, pod := range pods.Items {
			var payload string
			if pod.Name == hostname {
				ctx.Infof("adding leader metadata from pod: %s", pod.Name)
				payload = `{"metadata":{"labels":{"leader":"true"}}}`
			} else {
				ctx.Infof("removing leader metadata from pod: %s", pod.Name)
				payload = `{"metadata":{"labels":{"leader": null}}}`
			}

			_, err := ctx.Kubernetes().CoreV1().Pods(ctx.GetNamespace()).Patch(ctx,
				pod.Name,
				types.MergePatchType,
				[]byte(payload),
				metav1.PatchOptions{})
			if err != nil {
				return retry.RetryableError(err)
			}
		}

		return nil
	})
	if err != nil {
		ctx.Errorf("failed to set label: %v", err)
	}
}

// getAllReplicas returns all the pods from its parent ReplicaSet
func getAllReplicas(ctx context.Context, thisPod string) (*corev1.PodList, error) {
	pod, err := ctx.Kubernetes().CoreV1().Pods(ctx.GetNamespace()).Get(ctx, thisPod, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Get the ReplicaSet owner reference
	var replicaSetName string
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "ReplicaSet" {
			replicaSetName = ownerRef.Name
			break
		}
	}

	if replicaSetName == "" {
		return nil, errors.New("this pod is not managed by a ReplicaSet")
	}

	// List all pods with the same ReplicaSet label
	labelSelector := fmt.Sprintf("pod-template-hash=%s", pod.Labels["pod-template-hash"])
	podList, err := ctx.Kubernetes().CoreV1().Pods(ctx.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods with labelSelector(%s): %w", labelSelector, err)
	}

	return podList, nil
}
