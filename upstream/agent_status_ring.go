package upstream

import (
	"fmt"
	"sync"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

// StatusRingManager manages status rings for agent jobs.
type StatusRingManager interface {
	// Add adds the given history to the corresponding status ring of the agent.
	// if the status ring doesn't exist then it creates a new one.
	Add(ctx context.Context, agentID string, history models.JobHistory)
}

type simpleStatusRingManager struct {
	m           sync.Mutex
	evicted     chan uuid.UUID
	statusRings map[string]*job.StatusRing
}

func (t *simpleStatusRingManager) Add(ctx context.Context, agentID string, history models.JobHistory) {
	ring := t.getOrCreateRing(ctx, agentID, history)
	ring.Add(&history)
}

func (t *simpleStatusRingManager) key(agentID string, history models.JobHistory) string {
	return fmt.Sprintf("%s-%s-%s", agentID, history.Name, history.ResourceID)
}

func (t *simpleStatusRingManager) getOrCreateRing(ctx context.Context, agentID string, history models.JobHistory) *job.StatusRing {
	t.m.Lock()
	defer t.m.Unlock()

	key := t.key(agentID, history)
	if ring, ok := t.statusRings[key]; ok {
		return ring
	}

	// By default use a balanced retention
	retention := job.RetentionBalanced

	// Use retention from the properties if available
	dummyJob := job.NewJob(ctx, history.Name, "", nil)
	retention.Success = dummyJob.GetPropertyInt("retention.success", retention.Success)
	retention.Failed = dummyJob.GetPropertyInt("retention.failed", retention.Failed)

	ring := job.NewStatusRing(retention, false, t.evicted)
	t.statusRings[key] = &ring
	return &ring
}

func NewStatusRingStore(evicted chan uuid.UUID) StatusRingManager {
	return &simpleStatusRingManager{
		evicted:     evicted,
		statusRings: make(map[string]*job.StatusRing),
	}
}
