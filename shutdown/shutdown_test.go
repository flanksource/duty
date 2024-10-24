package shutdown

import "testing"

func TestShutdownPriority(t *testing.T) {
	var lastClosed int

	add := func(label string, priority int) {
		AddHookWithPriority(label, priority, func() {
			if lastClosed > priority {
				t.Fatalf("something higher priority (%d) was closed earlier than (%d)", lastClosed, priority)
			} else {
				lastClosed = priority
			}
		})
	}

	add("database", PriorityCritical)
	add("gRPC", PriorityIngress)
	add("checkJob", PriorityJobs)
	add("echo", PriorityIngress)
	add("topologyJob", PriorityJobs)

	Shutdown()
}
