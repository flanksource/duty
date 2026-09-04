package context

import (
	gocontext "context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flanksource/commons/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
)

// The env cache keeps Secret, ConfigMap, and Helm values in go-cache while
// namespace-scoped metadata informers provide invalidation. Informers are
// created lazily and remain active for the process lifetime; the longer cache
// timeout is used only when a fetched resource version matches the informer.
var (
	secretGVR    = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

	envCacheInformersMu          sync.Mutex
	envCacheInformersByNamespace = make(map[string]*cachedNamespaceInformerSet)
	noEnvCacheInformers          = &cachedNamespaceInformerSet{}

	// Serializing cache writes with invalidation prevents a stale GET from
	// repopulating an entry after the informer has processed its update.
	envCacheInvalidationMu sync.Mutex
)

// cachedNamespaceInformerSet owns the long-lived Secret and ConfigMap metadata
// informers responsible for invalidating env cache entries in one namespace.
type cachedNamespaceInformerSet struct {
	secret    *cacheInvalidationInformer
	configMap *cacheInvalidationInformer
}

// cacheInvalidationInformer wraps one resource-specific metadata informer and
// tracks whether it can safely support long-lived cache entries.
type cacheInvalidationInformer struct {
	informer    toolscache.SharedIndexInformer
	unavailable atomic.Bool
	stopCh      chan struct{}
	stopOnce    sync.Once
	warnOnce    sync.Once
}

func (i *cacheInvalidationInformer) ready() bool {
	return i != nil && !i.unavailable.Load() && i.informer.HasSynced()
}

func (i *cacheInvalidationInformer) start() {
	if i != nil && !i.unavailable.Load() {
		go i.informer.Run(i.stopCh)
	}
}

func (i *cacheInvalidationInformer) stop() {
	i.stopOnce.Do(func() { close(i.stopCh) })
}

func (i *cacheInvalidationInformer) warn(namespace, resource string, err error) {
	i.warnOnce.Do(func() {
		warnEnvCacheInformer(namespace, resource, err)
	})
}

func warnEnvCacheInformer(namespace, resource string, err error) {
	logger.GetLogger("envvar-cache").Warnf("metadata informer for Kubernetes %s in namespace %q reported an error: %v; env var cache entries without watch invalidation use the fallback timeout", resource, namespace, err)
}

func getCachedNamespaceInformerSet(namespace string, config *rest.Config) *cachedNamespaceInformerSet {
	if namespace == "" {
		return noEnvCacheInformers
	}

	envCacheInformersMu.Lock()
	defer envCacheInformersMu.Unlock()

	if informerSet, ok := envCacheInformersByNamespace[namespace]; ok {
		return informerSet
	}

	informerSet := &cachedNamespaceInformerSet{}
	envCacheInformersByNamespace[namespace] = informerSet
	if config == nil {
		warnEnvCacheInformer(namespace, "Secrets and ConfigMaps", fmt.Errorf("Kubernetes REST config is nil"))
		return informerSet
	}

	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		warnEnvCacheInformer(namespace, "Secrets and ConfigMaps", err)
		return informerSet
	}

	factory := metadatainformer.NewFilteredSharedInformerFactory(metadataClient, 0, namespace, nil)
	informerSet.secret = configureEnvCacheInformer(
		namespace,
		"Secrets",
		factory.ForResource(secretGVR).Informer(),
		metadataInvalidationHandler(invalidateSecretEnvCache),
		[]string{secretEnvCacheKey(namespace, ""), helmEnvCacheKey(namespace, "")},
	)
	informerSet.configMap = configureEnvCacheInformer(
		namespace,
		"ConfigMaps",
		factory.ForResource(configMapGVR).Informer(),
		metadataInvalidationHandler(invalidateConfigMapEnvCache),
		[]string{configMapEnvCacheKey(namespace, "")},
	)
	informerSet.secret.start()
	informerSet.configMap.start()
	return informerSet
}

func configureEnvCacheInformer(
	namespace, resource string,
	informer toolscache.SharedIndexInformer,
	handler toolscache.ResourceEventHandler,
	cachePrefixes []string,
) *cacheInvalidationInformer {
	configured := &cacheInvalidationInformer{informer: informer, stopCh: make(chan struct{})}
	if _, err := informer.AddEventHandler(handler); err != nil {
		configured.unavailable.Store(true)
		configured.warn(namespace, resource, err)
		return configured
	}

	if err := informer.SetWatchErrorHandlerWithContext(func(_ gocontext.Context, _ *toolscache.Reflector, err error) {
		if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
			envCacheInvalidationMu.Lock()
			configured.unavailable.Store(true)
			// Purge all entries that depended on this informer before stopping it;
			// immutable entries will regain their minimum TTL when next loaded.
			deleteEnvCachePrefixes(cachePrefixes)
			envCacheInvalidationMu.Unlock()
			configured.stop()
			warnEnvCacheInformer(namespace, resource, err)
			return
		}
		if !configured.ready() {
			configured.warn(namespace, resource, err)
		}
	}); err != nil {
		configured.unavailable.Store(true)
		configured.warn(namespace, resource, err)
	}
	return configured
}

// metadataInvalidationHandler includes initial-list additions because lookups
// may have populated the fallback cache while the informer was synchronizing.
func metadataInvalidationHandler(invalidate func(metav1.Object)) toolscache.ResourceEventHandler {
	invalidateObjects := func(objects ...any) {
		envCacheInvalidationMu.Lock()
		defer envCacheInvalidationMu.Unlock()
		for _, object := range objects {
			if metadata, ok := envCacheObjectMetadata(object); ok {
				invalidate(metadata)
			}
		}
	}

	return toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(object any) {
			invalidateObjects(object)
		},
		UpdateFunc: func(oldObject, newObject any) {
			oldMetadata, oldOK := envCacheObjectMetadata(oldObject)
			newMetadata, newOK := envCacheObjectMetadata(newObject)
			if !oldOK || !newOK || oldMetadata.GetResourceVersion() == newMetadata.GetResourceVersion() {
				return
			}
			invalidateObjects(oldObject, newObject)
		},
		DeleteFunc: func(object any) {
			invalidateObjects(object)
		},
	}
}

func envCacheObjectMetadata(object any) (metav1.Object, bool) {
	switch tombstone := object.(type) {
	case toolscache.DeletedFinalStateUnknown:
		object = tombstone.Obj
	case *toolscache.DeletedFinalStateUnknown:
		object = tombstone.Obj
	}
	metadata, err := apimeta.Accessor(object)
	return metadata, err == nil
}

func invalidateSecretEnvCache(metadata metav1.Object) {
	envCache.Delete(secretEnvCacheKey(metadata.GetNamespace(), metadata.GetName()))
	if releaseName := metadata.GetLabels()["name"]; releaseName != "" {
		envCache.Delete(helmEnvCacheKey(metadata.GetNamespace(), releaseName))
	}
}

func invalidateConfigMapEnvCache(metadata metav1.Object) {
	envCache.Delete(configMapEnvCacheKey(metadata.GetNamespace(), metadata.GetName()))
}

func deleteEnvCachePrefixes(prefixes []string) {
	for key := range envCache.Items() {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				envCache.Delete(key)
				break
			}
		}
	}
}

func cacheEnvObject(
	informer *cacheInvalidationInformer,
	namespace, name, resourceVersion, cacheKey string,
	value any,
	fallbackTimeout, watchedTimeout time.Duration,
) {
	envCacheInvalidationMu.Lock()
	defer envCacheInvalidationMu.Unlock()

	timeout := fallbackTimeout
	if informer.ready() {
		object, exists, err := informer.informer.GetStore().GetByKey(namespace + "/" + name)
		if err != nil || !exists {
			return
		}
		metadata, ok := envCacheObjectMetadata(object)
		if !ok || metadata.GetResourceVersion() != resourceVersion {
			return
		}
		timeout = watchedTimeout
	}
	envCache.Set(cacheKey, value, timeout)
}
