package shutdown

import (
	"container/heap"
	"os"
	"os/signal"
	"sync"

	"github.com/flanksource/commons/logger"
)

// Some helper priority levels
const (
	PriorityIngress  = 100
	PriorityJobs     = 500
	PriorityCritical = 1000
)

type shutdownHook func()

var m sync.Mutex

var shutdownTaskRegistry ShutdownTasks

func init() {
	heap.Init(&shutdownTaskRegistry)
}

var Shutdown = sync.OnceFunc(func() {
	logger.Infof("shutting down")
	for _, task := range shutdownTaskRegistry {
		logger.Infof("shutting down: %s", task.Label)
		task.Hook()
	}
})

func ShutdownAndExit(code int, msg string) {
	Shutdown()
	logger.StandardLogger().WithSkipReportLevel(1).Errorf(msg)
	os.Exit(code)
}

// @Deprecated
// Prefer AddHookWithPriority()
func AddHook(fn shutdownHook) {
	m.Lock()
	heap.Push(&shutdownTaskRegistry, ShutdownTask{Hook: fn, Priority: 0})
	m.Unlock()
}

// Execution order goes from lowest to highest priority numbers.
func AddHookWithPriority(label string, priority int, fn shutdownHook) {
	m.Lock()
	heap.Push(&shutdownTaskRegistry, ShutdownTask{Label: label, Hook: fn, Priority: priority})
	m.Unlock()
}

func WaitForSignal() {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		logger.Infof("Caught Ctrl+C")
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

// Less defines higher priority numbers will be processed first
func (st ShutdownTasks) Less(i, j int) bool {
	return st[i].Priority > st[j].Priority // Higher priority numbers come first
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
