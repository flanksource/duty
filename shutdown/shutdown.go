package shutdown

import (
	"container/heap"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/flanksource/commons/logger"
)

// Some helper priority levels
const (
	PriorityIngress  = 100
	PriorityJobs     = 500
	PriorityCritical = 1000
)

type shutdownHook func()

var (
	registryLock         sync.Mutex
	shutdownTaskRegistry ShutdownTasks
)

func init() {
	heap.Init(&shutdownTaskRegistry)
}

var Shutdown = sync.OnceFunc(func() {
	logger.Infof("begin shutdown")

	for len(shutdownTaskRegistry) > 0 {
		_task := heap.Pop(&shutdownTaskRegistry)
		if _task == nil {
			break
		}

		task := _task.(ShutdownTask)
		logger.Infof("shutting down: %s", task.Label)

		s := time.Now()
		task.Hook()
		logger.Infof("shutdown %s completed in %v", task.Label, time.Since(s))
	}
})

func ShutdownAndExit(code int, msg string) {
	if code == 0 {
		logger.StandardLogger().WithSkipReportLevel(1).Infof(msg)
	} else {
		logger.StandardLogger().WithSkipReportLevel(1).Errorf(msg)
	}

	Shutdown()
	os.Exit(code)
}

// Add a hook with the least priority.
// Least priority hooks are run first.
//
// Prefer AddHookWithPriority()
func AddHook(fn shutdownHook) {
	registryLock.Lock()
	heap.Push(&shutdownTaskRegistry, ShutdownTask{Hook: fn, Priority: 0})
	registryLock.Unlock()
}

// AddHookWithPriority adds a hook with a priority level.
//
// Execution order goes from lowest to highest priority numbers.
func AddHookWithPriority(label string, priority int, fn shutdownHook) {
	registryLock.Lock()
	heap.Push(&shutdownTaskRegistry, ShutdownTask{Label: label, Hook: fn, Priority: priority})
	registryLock.Unlock()
}

func WaitForSignal() {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		sig := <-quit
		logger.Infof("caught signal: %s", sig)
		// call shutdown hooks explicitly, post-run cleanup hooks will be a no-op
		Shutdown()
	}()
}

type ShutdownTask struct {
	Hook     shutdownHook
	Label    string
	Priority int
}

// ShutdownTasks implements heap.Interface
type ShutdownTasks []ShutdownTask

func (st ShutdownTasks) Len() int { return len(st) }

func (st ShutdownTasks) Less(i, j int) bool {
	return st[i].Priority < st[j].Priority
}

func (st ShutdownTasks) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

func (st *ShutdownTasks) Push(x interface{}) {
	*st = append(*st, x.(ShutdownTask))
}

func (st *ShutdownTasks) Pop() interface{} {
	old := *st
	n := len(old)
	item := old[n-1]
	*st = old[0 : n-1]
	return item
}
