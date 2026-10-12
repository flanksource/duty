package rbac

import (
	"net/http"
	"testing"

	"github.com/flanksource/duty/rbac/policy"
	"github.com/onsi/gomega"
)

func TestJobHistoryAndEventQueueObjects(t *testing.T) {
	for table, object := range map[string]string{
		"job_history":               policy.ObjectJobs,
		"job_histories":             policy.ObjectJobs,
		"job_history_names":         policy.ObjectJobs,
		"job_history_summary":       policy.ObjectJobs,
		"job_history_latest_status": policy.ObjectJobs,
		"event_queue":               policy.ObjectEvents,

		// A resource's latest job status is visible with the resource.
		"canaries_with_status":        policy.ObjectCanary,
		"topologies_with_status":      policy.ObjectTopology,
		"teams_with_status":           policy.ObjectDatabasePublic,
		"config_scrapers_with_status": policy.ObjectMonitor,
		"integrations_with_status":    policy.ObjectMonitor,
		"notifications_summary":       policy.ObjectMonitor,

		// Reading properties stays open; property:manage only gates writes.
		"properties": policy.ObjectDatabaseSystem,
	} {
		t.Run(table, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(GetObjectByTable(table)).To(gomega.Equal(object))
		})
	}
}

func TestReadIsTheActionOfGet(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(GetActionFromHttpMethod(http.MethodGet)).To(gomega.Equal(policy.ActionRead))
}

func TestActionsWithoutResource(t *testing.T) {
	g := gomega.NewWithT(t)

	g.Expect(policy.ObjectJobs).To(gomega.Equal("jobs"))
	g.Expect(policy.ActionJobRead).To(gomega.Equal("job:read"))
	g.Expect(policy.ObjectEvents).To(gomega.Equal("events"))
	g.Expect(policy.ActionEventRead).To(gomega.Equal("event:read"))
	g.Expect(policy.ObjectProperties).To(gomega.Equal("properties"))
	g.Expect(policy.ActionPropertyManage).To(gomega.Equal("property:manage"))

	g.Expect(policy.AllObjects).To(gomega.ContainElements(policy.ObjectJobs, policy.ObjectEvents, policy.ObjectProperties))
	g.Expect(policy.AllActions).To(gomega.ContainElements(policy.ActionJobRead, policy.ActionEventRead, policy.ActionPropertyManage))
}
