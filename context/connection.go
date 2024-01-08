package context

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("NOT_FOUND")
)

// extractConnectionNameType extracts the name and connection type from a connection
// string formatted as "connection://<type>/<namespace>/<name>".
func extractConnectionNameType(connectionString string) (name, namespace, connectionType string, found bool) {
	prefix := "connection://"

	if !strings.HasPrefix(connectionString, prefix) {
		return
	}

	connectionString = strings.TrimPrefix(connectionString, prefix)
	parts := strings.Split(connectionString, "/")
	if len(parts) > 3 || len(parts) < 2 {
		return
	}

	if parts[0] == "" || parts[1] == "" {
		return
	}

	if len(parts) == 3 {
		name, namespace, connectionType = parts[2], parts[1], parts[0]
		return name, namespace, connectionType, true
	} else if len(parts) == 2 {
		name, connectionType = parts[1], parts[0]
		return name, "", connectionType, true
	}

	return
}

// HydrateConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be in one of the following forms:
//   - connection://<type>/<name> or connection://<type>/<namespace>/<name>
//   - the UUID of the connection.
func HydrateConnectionByURL(ctx Context, connectionString string) (*models.Connection, error) {
	if connectionString == "" {
		return nil, nil
	}

	// Must be in one of the correct forms.
	if _, err := uuid.Parse(connectionString); err != nil && !strings.HasPrefix(connectionString, "connection://") {
		if _url, err := url.Parse(connectionString); err == nil {
			return models.ConnectionFromURL(*_url), nil
		}

		return nil, fmt.Errorf("invalid connection string: %q. Expected connection://<type>/<name> , uuid or URL", connectionString)
	}

	connection, err := FindConnectionByURL(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (%s): %w", connectionString, err)
	}

	if connection == nil {
		return nil, nil
	}

	return HydrateConnection(ctx, connection)
}

func IsValidConnectionURL(connectionString string) bool {
	if _, err := uuid.Parse(connectionString); err == nil {
		return true
	}
	_, _, _, found := extractConnectionNameType(connectionString)
	return found
}

// FindConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be of the form: connection://<type>/<name>
func FindConnectionByURL(ctx Context, connectionString string) (*models.Connection, error) {
	if _, err := uuid.Parse(connectionString); err == nil {
		var connection models.Connection
		if err := ctx.DB().Where("id = ?", connectionString).First(&connection).Error; err != nil {
			return nil, err
		}
		return &connection, nil
	}

	name, namespace, connectionType, found := extractConnectionNameType(connectionString)
	if !found {
		return nil, nil
	}

	connection, err := FindConnection(ctx, connectionType, name, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (type=%s, name=%s, namespace=%s): %w", connectionType, name, namespace, err)
	}

	return connection, nil
}

// FindConnection returns the connection with the given type and name
func FindConnection(ctx Context, connectionType, name, namespace string) (*models.Connection, error) {
	var connection models.Connection

	if namespace == "" {
		namespace = ctx.GetNamespace()
	}

	err := ctx.DB().Where("type = ? AND name = ? AND namespace = ?", connectionType, name, namespace).First(&connection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &connection, nil
}

func (ctx Context) GetConnection(connectionType, name, namespace string) (*models.Connection, error) {
	return GetConnection(ctx, connectionType, name, namespace)
}

func GetConnection(ctx Context, connectionType, name, namespace string) (*models.Connection, error) {
	connection, err := FindConnection(ctx, connectionType, name, namespace)
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
	if connection.Username, err = GetEnvStringFromCache(ctx, connection.Username, connection.Namespace); err != nil {
		return nil, err
	}

	if connection.Password, err = GetEnvStringFromCache(ctx, connection.Password, connection.Namespace); err != nil {
		return nil, err
	}

	if connection.Certificate, err = GetEnvStringFromCache(ctx, connection.Certificate, connection.Namespace); err != nil {
		return nil, err
	}

	domain := ""
	parts := strings.Split(connection.Username, "@")
	if len(parts) == 2 {
		domain = parts[1]
	}

	data := map[string]interface{}{
		"name":      connection.Name,
		"type":      connection.Type,
		"namespace": connection.Namespace,
		"username":  connection.Username,
		"password":  connection.Password,
		"domain":    domain,
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
