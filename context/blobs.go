package context

import (
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/artifact"
)

var blobsLogger = logger.GetLogger("blobs")

// BlobStoreProvider resolves a connection URL and returns a BlobStore backed by an external FS.
// It is set by the connection package during init().
var BlobStoreProvider func(ctx Context, connURL string) (artifact.BlobStore, error)

// Blobs returns the appropriate blob store for this context.
// If an artifacts.connection property is configured and a provider is registered,
// it returns the external backend. Otherwise it returns the inline DB-backed store.
func (k Context) Blobs() (artifact.BlobStore, error) {
	connURL := k.Properties().String("artifacts.connection", "")
	if connURL == "" {
		blobsLogger.Infof("Initializing inline blob store")
		store := artifact.NewBlobStore(artifact.NewInlineStore(k.DB()), k.DB(), "inline")
		return artifact.NewLoggedBlobStore(store, blobsLogger, "inline"), nil
	}
	if BlobStoreProvider == nil {
		return nil, fmt.Errorf("artifacts.connection is configured as %q but no blob store provider is registered", connURL)
	}
	return BlobStoreProvider(k, connURL)
}
