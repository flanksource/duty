package context

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	gocache "github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/models"
)

var (
	ErrNotFound = errors.New("NOT_FOUND")
)

// extractConnectionNameType extracts the name and connection type from a connection
// string formatted as
//
//	(Deprecated) "connection://<type>/<namespace>/<name>").
//	"connection://<namespace>/<name>".
//	"connection://<name>".
func extractConnectionNameType(connectionString string) (name, namespace string, found bool) {
	prefix := "connection://"

	if !strings.HasPrefix(connectionString, prefix) {
		return
	}

	found = true

	connectionString = strings.TrimPrefix(connectionString, prefix)
	parts := strings.Split(connectionString, "/")
	parts = lo.Map(parts, func(item string, _ int) string {
		return strings.TrimSpace(item)
	})

	switch len(parts) {
	case 3:
		name, namespace = parts[2], parts[1]

	case 2:
		name, namespace = parts[1], parts[0]

	case 1:
		name = parts[0]

	default:
		found = false
	}

	// namespace can be left unspecified but name is mandatory.
	if name == "" {
		found = false
	}

	return
}

var connectionCache = cache.New[*models.Connection](gocache_store.NewGoCache(gocache.New(30*time.Minute, 30*time.Minute)))

func getConnectionCacheKey(connString, ctxNamespace string) string {
	return connString + ctxNamespace
}

// HydrateConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be in one of the following forms:
//   - connection://<namespace>/<name> or connection://<name>
//   - the UUID of the connection.
func HydrateConnectionByURL(ctx Context, connectionString string) (*models.Connection, error) {
	if connectionString == "" {
		return nil, nil
	}

	cacheKey := getConnectionCacheKey(connectionString, ctx.GetNamespace())
	if cacheVal, err := connectionCache.Get(ctx, cacheKey); err == nil {
		return cacheVal, nil
	}

	_, uuidErr := uuid.Parse(connectionString)
	isConnectionUUID := uuidErr == nil

	if !isConnectionUUID && !strings.HasPrefix(connectionString, "connection://") {
		if _url, err := url.Parse(connectionString); err == nil {
			return models.ConnectionFromURL(*_url), nil
		}
	}

	_, _, formatOK := extractConnectionNameType(connectionString)
	if !formatOK && !isConnectionUUID {
		// Must be in one of the correct forms.
		return nil, fmt.Errorf("invalid connection string: %q. Expected connection string (connection://<namespace>/<name>), uuid or a URL", connectionString)
	}

	connection, err := FindConnectionByURL(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (%s): %w", connectionString, err)
	}

	if connection == nil {
		// Setting a smaller cache for connection not found
		_ = connectionCache.Set(ctx, cacheKey, connection, store.WithExpiration(5*time.Minute))
		return nil, fmt.Errorf("connection %q not found", connectionString)
	}

	hydratedConnection, err := HydrateConnection(ctx, connection)
	if err == nil {
		_ = connectionCache.Set(ctx, cacheKey, hydratedConnection)
	}
	return hydratedConnection, err
}

func IsValidConnectionURL(connectionString string) bool {
	if _, err := uuid.Parse(connectionString); err == nil {
		return true
	}
	_, _, found := extractConnectionNameType(connectionString)
	return found
}

// FindConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be in one of the following forms:
//   - connection://<namespace>/<name> or connection://<name>
//   - the UUID of the connection.
func FindConnectionByURL(ctx Context, connectionString string) (*models.Connection, error) {
	db := ctx.DB()

	if db == nil {
		return nil, fmt.Errorf("db is not configured")
	}

	if _, err := uuid.Parse(connectionString); err == nil {
		var connection models.Connection
		if err := db.Where("id = ?", connectionString).First(&connection).Error; err != nil {
			return nil, err
		}
		return &connection, nil
	}

	name, namespace, found := extractConnectionNameType(connectionString)
	if !found {
		return nil, fmt.Errorf("invalid connection string: %q. Must be in connection://<namespace>/<name> format", connectionString)
	}

	connection, err := FindConnection(ctx, name, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (name=%s, namespace=%s): %w", name, namespace, err)
	}

	return connection, nil
}

// FindConnection returns the connection with the given type and name
func FindConnection(ctx Context, name, namespace string) (*models.Connection, error) {
	var connection models.Connection

	if namespace == "" {
		namespace = ctx.GetNamespace()
	}

	db := ctx.DB()

	if db == nil {
		return nil, fmt.Errorf("db is not configured")
	}

	if err := db.Where("name = ? AND namespace = ? AND deleted_at IS NULL", name, namespace).
		First(&connection).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if connection.ID == uuid.Nil {
		// NOTE: For backward compatibility reason we use the namespace as the connection type
		// Before: connection://<type>/<name>
		// Now: connection://<namespace>/<name.
		if err := ctx.DB().Where("name = ? AND type = ? AND deleted_at IS NULL", name, namespace).
			First(&connection).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else if connection.ID != uuid.Nil {
			logger.Warnf("connection format connection://<type>/<name> has been deprecated. Use connection://<namespace>/<name> or connection://<name>")
		} else if errors.Is(err, gorm.ErrRecordNotFound) || connection.ID == uuid.Nil {
			// The connection does not exist either way
			return nil, nil
		}
	}

	return &connection, nil
}

func (ctx Context) GetConnection(name, namespace string) (*models.Connection, error) {
	return GetConnection(ctx, name, namespace)
}

func GetConnection(ctx Context, name, namespace string) (*models.Connection, error) {
	connection, err := FindConnection(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	if connection == nil {
		return nil, ErrNotFound
	}

	return HydrateConnection(ctx, connection)
}

// Create a cache with a default expiration time of 5 minutes, and which
// purges expired items every 10 minutes
// var connectionCache = cache.New(5*time.Minute, 10*time.Minute)
func HydrateConnection(ctx Context, connection *models.Connection) (*models.Connection, error) {
	var err error

	if connection.URL, err = GetEnvStringFromCache(ctx, connection.URL, connection.Namespace); err != nil {
		return nil, err
	}

	if connection.Username, err = GetEnvStringFromCache(ctx, connection.Username, connection.Namespace); err != nil {
		return nil, err
	}

	if connection.Password, err = GetEnvStringFromCache(ctx, connection.Password, connection.Namespace); err != nil {
		return nil, err
	}

	if connection.Certificate, err = GetEnvStringFromCache(ctx, connection.Certificate, connection.Namespace); err != nil {
		return nil, err
	}

	for k, v := range connection.Properties {
		if v, err = GetEnvStringFromCache(ctx, v, connection.Namespace); err != nil {
			return nil, err
		} else {
			connection.Properties[k] = v
		}
	}

	// Remove newlines and spaces around username,password & url
	connection.URL = strings.TrimSpace(connection.URL)
	connection.Username = strings.TrimSpace(connection.Username)
	connection.Password = strings.TrimSpace(connection.Password)

	domain := ""
	parts := strings.Split(connection.Username, "@")
	if len(parts) == 2 {
		domain = parts[1]
	}

	data := map[string]interface{}{
		"name":       connection.Name,
		"type":       connection.Type,
		"namespace":  connection.Namespace,
		"username":   url.QueryEscape(connection.Username),
		"password":   url.QueryEscape(connection.Password),
		"domain":     domain,
		"properties": connection.Properties,
	}
	templater := gomplate.StructTemplater{
		Values: data,
		// access go values in template requires prefix everything with .
		// to support $(username) instead of $(.username) we add a function for each var
		ValueFunctions: true,
		DelimSets: []gomplate.Delims{
			{Left: "{{", Right: "}}"},
			{Left: "$(", Right: ")"},
		},
		RequiredTag: "template",
	}
	if err := templater.Walk(connection); err != nil {
		return nil, err
	}

	return connection, nil
}
