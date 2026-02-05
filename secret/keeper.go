package secret

import (
	"fmt"
	"sync"
	"time"

	"github.com/flanksource/duty/connection"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"github.com/samber/oops"
	"gocloud.dev/secrets"
)

const defaultKeeperTTL = time.Minute * 10

var (
	keeperCache = cache.New(defaultKeeperTTL, defaultKeeperTTL*2)

	// keeperLock locks access to the keeperCache
	keeperLock sync.RWMutex
)

var (
	// KMSConnection is the connection to the key management service
	// that's used to encrypt and decrypt secrets.
	KMSConnection string

	allowedConnectionTypes = []string{
		models.ConnectionTypeAWSKMS,
		models.ConnectionTypeGCPKMS,
		models.ConnectionTypeAzureKeyVault,
		// Vault not supported yet
	}
)

func init() {
	keeperCache.OnEvicted(func(key string, keeper interface{}) {
		if keeper != nil {
			keeper.(*secrets.Keeper).Close()
		}
	})
}

// createOrGetKeeper creates a new Keeper from the KMSConnection if it doesn't
// exist in the cache, otherwise it returns the cached Keeper.
func createOrGetKeeper(ctx context.Context) (*secrets.Keeper, error) {
	if KMSConnection == "" {
		return nil, oops.Errorf("secret keeper connection is not set")
	}

	keeperLock.RLock()
	cached, ok := keeperCache.Get("keeper")
	keeperLock.RUnlock()
	if ok {
		return cached.(*secrets.Keeper), nil
	}

	keeperLock.Lock()
	defer keeperLock.Unlock()

	keeper, err := KeeperFromConnection(ctx, KMSConnection)
	if err != nil {
		return nil, err
	}

	ttl := ctx.Properties().Duration("secretkeeper.cache.ttl", defaultKeeperTTL)
	keeperCache.Set("keeper", keeper, ttl)
	return keeper, nil
}

func KeeperFromConnection(ctx context.Context, connectionString string) (*secrets.Keeper, error) {
	conn, err := ctx.HydrateConnectionByURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to hydrate connection: %w", err)
	} else if conn == nil {
		return nil, fmt.Errorf("connection not found: %s", connectionString)
	}

	if !lo.Contains(allowedConnectionTypes, conn.Type) {
		return nil, fmt.Errorf("connection type %s cannot be used to create a SecretKeeper", conn.Type)
	}

	switch conn.Type {
	case models.ConnectionTypeAWSKMS:
		var kmsConn connection.AWSKMS
		kmsConn.FromModel(*conn)
		return kmsConn.SecretKeeper(ctx)

	case models.ConnectionTypeAzureKeyVault:
		var keyvaultConn connection.AzureKeyVault
		keyvaultConn.FromModel(*conn)
		return keyvaultConn.SecretKeeper(ctx)

	case models.ConnectionTypeGCPKMS:
		var kmsConn connection.GCPKMS
		kmsConn.FromModel(*conn)
		return kmsConn.SecretKeeper(ctx)
	}

	return nil, nil
}
