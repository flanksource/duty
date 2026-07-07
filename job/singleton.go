package job

import (
	"sync"
)

var singletonLocks = newSingletonLockRegistry()

type singletonLockRegistry struct {
	mu      sync.Mutex
	running map[string]struct{}
}

func newSingletonLockRegistry() *singletonLockRegistry {
	return &singletonLockRegistry{
		running: make(map[string]struct{}),
	}
}

func (r *singletonLockRegistry) TryLock(key string) (func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.running[key]; ok {
		return nil, false
	}

	r.running[key] = struct{}{}

	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			delete(r.running, key)
		})
	}, true
}
