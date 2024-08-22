package cache

import (
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
)

func NewCache[T any](name string, expiration time.Duration) cache.CacheInterface[T] {
	return cache.New[T](gocache_store.NewGoCache(gocache.New(expiration, expiration)))
}
