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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/echo"
	"github.com/flanksource/duty/shutdown"
)

var (
	hostname     string
	podNamespace string
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
}

func Register(
	ctx context.Context,
	app string,
	namespace string,
	onLead func(ctx gocontext.Context),
	onStoppedLead func(),
	onNewLeader func(identity string),
) error {
	if namespace == "" {
		namespace = podNamespace
	}

	ctx = ctx.WithNamespace(namespace)
	client, err := ctx.LocalKubernetes()
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      app,
			Namespace: namespace,
		},
		Client: client.CoordinationV1(),
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
				ctx.Infof("started leading election")

				updateLeaderLabel(ctx, app)

				for entry := range echo.Crons.IterBuffered() {
					entry.Val.Start()
				}

				if onLead != nil {
					onLead(leadCtx)
				}
			},
			OnStoppedLeading: func() {
				ctx.Infof("stopped leading election")

				for entry := range echo.Crons.IterBuffered() {
					entry.Val.Stop()
				}

				if onStoppedLead != nil {
					onStoppedLead()
				}
			},
			OnNewLeader: func(identity string) {
				if identity == hostname {
					return
				}

				if onNewLeader != nil {
					onNewLeader(identity)
				}
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

	go func() {
		// when a network failure occurs for a considerable amount of time (>30s)
		// elector.Run terminates and never retries acquiring the lease.
		//
		// that's why it's run in a never ending loop
		for {
			select {
			case <-leaderContext.Done():
				return
			default:
				elector.Run(leaderContext)
			}
		}
	}()
	<-ctx.Done()

	return nil
}

// updateLeaderLabel sets leader:true label on the current pod
// and also removes that label from all other replicas.
func updateLeaderLabel(ctx context.Context, app string) {
	backoff := retry.WithMaxRetries(3, retry.NewExponential(time.Second))
	err := retry.Do(ctx, backoff, func(_ctx gocontext.Context) error {
		labelSelector := fmt.Sprintf("%s/leader=true", app)
		client, err := ctx.LocalKubernetes()
		if err != nil {
			return fmt.Errorf("error creating kubernetes client: %w", err)
		}
		podList, err := client.CoreV1().Pods(ctx.GetNamespace()).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return retry.RetryableError(fmt.Errorf("failed to list pods with labelSelector(%s): %w", labelSelector, err))
		}

		pods := lo.Map(podList.Items, func(p corev1.Pod, _ int) string { return p.Name })
		pods = append(pods, hostname)

		for _, podName := range lo.Uniq(pods) {
			var payload string
			if podName == hostname {
				ctx.Infof("adding leader metadata to pod: %s", podName)
				payload = fmt.Sprintf(`{"metadata":{"labels":{"%s/leader":"true"}}}`, app)
			} else {
				ctx.Infof("removing leader metadata to pod: %s", podName)
				payload = fmt.Sprintf(`{"metadata":{"labels":{"%s/leader": null}}}`, app)
			}

			_, err = client.CoreV1().Pods(ctx.GetNamespace()).Patch(ctx,
				podName,
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
