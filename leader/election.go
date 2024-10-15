package leader

import (
	gocontext "context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/sethvargo/go-retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/flanksource/duty/context"
)

var (
	hostname string
	service  string

	// namespace the pod is running on
	namespace string
)

const namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func podNamespace() (string, error) {
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

	if n, err := podNamespace(); err != nil {
		log.Fatalf("failed to get pod namespace: %v", err)
	} else {
		namespace = n
	}

	service = strings.Split(hostname, "-")[0]
}

func Register(
	ctx context.Context,
	onLead func(ctx gocontext.Context),
	onStoppedLead func(),
	onNewLeader func(identity string),
) {
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

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   ctx.Properties().Duration("leader.lease.duration", 30*time.Second),
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leadCtx gocontext.Context) {
				updateLeaderLabel(ctx, true)
				onLead(leadCtx)
			},
			OnStoppedLeading: func() {
				updateLeaderLabel(ctx, false)
				onStoppedLead()
			},
			OnNewLeader: func(identity string) {
				if identity == hostname {
					return
				}

				onNewLeader(identity)
			},
		},
	})
}

func updateLeaderLabel(ctx context.Context, set bool) {
	payload := `{"metadata":{"labels":{"leader":"true"}}}`
	if !set {
		payload = `{"metadata":{"labels":{"leader": null}}}`
	}

	backoff := retry.WithMaxRetries(3, retry.NewExponential(time.Second))
	err := retry.Do(ctx, backoff, func(_ctx gocontext.Context) error {
		_, err := ctx.Kubernetes().CoreV1().Pods(namespace).Patch(ctx,
			hostname,
			types.MergePatchType,
			[]byte(payload),
			metav1.PatchOptions{})
		return retry.RetryableError(err)
	})
	if err != nil {
		ctx.Errorf("failed to %sset label", lo.Ternary(set, "", "un"))
	}
}
