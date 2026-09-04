package context

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	toolscache "k8s.io/client-go/tools/cache"
)

func TestCacheEnvObjectDoesNotRepopulateDuringInitialSync(t *testing.T) {
	g := gomega.NewWithT(t)
	previousCache := envCache
	envCache = cache.New(envCacheDefaultTimeout, 0)
	t.Cleanup(func() { envCache = previousCache })

	informer := toolscache.NewSharedIndexInformer(nil, &metav1.PartialObjectMetadata{}, 0, toolscache.Indexers{})
	configured := &cacheInvalidationInformer{informer: informer}
	g.Expect(informer.HasSynced()).To(gomega.BeFalse())
	key := secretEnvCacheKey("namespace", "secret")
	envCache.Set(key, "old", cache.NoExpiration)

	metadataInvalidationHandler(invalidateSecretEnvCache).OnAdd(&metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "namespace",
			Name:            "secret",
			ResourceVersion: "2",
		},
	}, true)
	_, found := envCache.Get(key)
	g.Expect(found).To(gomega.BeFalse())

	cacheEnvObject(configured, "namespace", "secret", "1", key, "stale", time.Minute, time.Hour)
	_, found = envCache.Get(key)
	g.Expect(found).To(gomega.BeFalse())
}
