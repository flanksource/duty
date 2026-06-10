package shutdown

import (
	"os"
	"os/signal"
	"sort"
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
	registryLock  sync.Mutex
	shutdownTasks []ShutdownTask
)

var Shutdown = sync.OnceFunc(func() {
	logger.Infof("begin shutdown")

	registryLock.Lock()
	tasks := shutdownTasks
	shutdownTasks = nil
	registryLock.Unlock()

	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
	})

	for _, task := range tasks {
		if task.Hook == nil {
			continue
		}
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
	shutdownTasks = append(shutdownTasks, ShutdownTask{Hook: fn, Priority: 0})
	registryLock.Unlock()
}

// AddHookWithPriority adds a hook with a priority level.
//
// Execution order goes from lowest to highest priority numbers.
func AddHookWithPriority(label string, priority int, fn shutdownHook) {
	registryLock.Lock()
	shutdownTasks = append(shutdownTasks, ShutdownTask{Label: label, Hook: fn, Priority: priority})
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
