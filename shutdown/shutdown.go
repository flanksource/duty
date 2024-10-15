package shutdown

import (
	"os"
	"os/signal"
	"sync"

	"github.com/flanksource/commons/logger"
)

var shutdownHooks []func()

var Shutdown = sync.OnceFunc(func() {
	if len(shutdownHooks) == 0 {
		return
	}
	logger.Infof("Shutting down")
	for _, fn := range shutdownHooks {
		fn()
	}
	shutdownHooks = []func(){}
})

func ShutdownAndExit(code int, msg string) {
	Shutdown()
	logger.StandardLogger().WithSkipReportLevel(1).Errorf(msg)
	os.Exit(code)
}

func AddHook(fn func()) {
	shutdownHooks = append(shutdownHooks, fn)
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
