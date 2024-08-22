package leader

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/flanksource/duty/context"
	"github.com/samber/lo"

	v1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	identity = getHostname()
)

var isLeader *atomic.Bool

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}
	return hostname
}

var watchers = []func(isLeader bool){}

func OnElection(ctx context.Context, leaseName string, fn func(isLeader bool)) {
	watchers = append(watchers, fn)

	if isLeader == nil {
		leaseDuration := ctx.Properties().Duration("leader.lease.duration", 10*time.Minute)
		isLeader = new(atomic.Bool)
		go func() {
			for {

				leader, _, err := createOrUpdateLease(ctx, leaseName, 0)

				if err != nil {
					ctx.Warnf("failed to create/update lease: %v", err)
					time.Sleep(time.Duration(rand.Intn(60)) * time.Second)
					continue
				}

				if isLeader.Load() != leader {
					isLeader.Store(leader)
					notify(leader)
				}

				// sleep for just under half the lease duration before trying to renew
				time.Sleep(leaseDuration/2 - time.Second*10)
			}
		}()
	}

}

func notify(isLeader bool) {
	for _, fn := range watchers {
		fn(isLeader)
	}

}

func IsLeader(ctx context.Context, leaseName string) (bool, error) {
	leases := ctx.Kubernetes().CoordinationV1().Leases(ctx.GetNamespace())

	lease, err := leases.Get(ctx, leaseName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if *lease.Spec.HolderIdentity == identity {
		return true, nil
	}

	return false, nil
}

func createOrUpdateLease(ctx context.Context, leaseName string, attempt int) (bool, string, error) {
	if attempt > 0 {
		time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
	}
	if attempt >= ctx.Properties().Int("leader.lease.attempts", 3) {
		return false, "", fmt.Errorf("failed to acquire lease %s after %d attempts", leaseName, attempt)
	}
	now := metav1.MicroTime{Time: time.Now()}
	leases := ctx.Kubernetes().CoordinationV1().Leases(ctx.GetNamespace())
	leaseDuration := ctx.Properties().Duration("leader.lease.duration", 10*time.Minute)
	lease, err := leases.Get(ctx, leaseName, metav1.GetOptions{})
	if err != nil {
		return false, "", err
	}
	if lease == nil {
		ctx.Infof("Acquiring lease %s", leaseName)
		lease = &v1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: ctx.GetNamespace(),
			},
			Spec: v1.LeaseSpec{
				HolderIdentity:       &identity,
				LeaseDurationSeconds: lo.ToPtr(int32(leaseDuration.Seconds())),
				AcquireTime:          &now,
				RenewTime:            &now,
			},
		}
		_, err = leases.Create(ctx, lease, metav1.CreateOptions{})
		if err != nil {
			return false, "", err
		}
		return true, identity, nil
	}

	if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == identity {
		lease.Spec.RenewTime = &now
		ctx.Debugf("Renewing lease %s : %s", leaseName, now.String())
		_, err = leases.Update(ctx, lease, metav1.UpdateOptions{})
		if err != nil {
			return false, "", err
		}
	}
	renewTime := lease.Spec.RenewTime.Time
	if time.Since(renewTime) > leaseDuration {
		ctx.Infof("Lease %s held by %s expired", leaseName, *lease.Spec.HolderIdentity)
		if err := leases.Delete(ctx, leaseName, metav1.DeleteOptions{}); err != nil {
			ctx.Infof("failed to delete leases %s: %v", leaseName, err)
		}
		return createOrUpdateLease(ctx, leaseName, attempt+1)
	}
	ctx.Debugf("Lease %s already held by %s, expires in %s", leaseName, *lease.Spec.HolderIdentity, time.Until(renewTime.Add(leaseDuration)).String())
	return false, *lease.Spec.HolderIdentity, nil
}
