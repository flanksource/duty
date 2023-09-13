package duty

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flanksource/commons/template"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrNotFound = errors.New("NOT_FOUND")
)

// extractConnectionNameType extracts the name and connection type from a connection
// string formatted as "connection://<type>/<name>".
func extractConnectionNameType(connectionString string) (name string, connectionType string, found bool) {
	prefix := "connection://"

	if !strings.HasPrefix(connectionString, prefix) {
		return
	}

	connectionString = strings.TrimPrefix(connectionString, prefix)
	parts := strings.SplitN(connectionString, "/", 2)
	if len(parts) != 2 {
		return
	}

	if parts[0] == "" || parts[1] == "" {
		return
	}

	return parts[1], parts[0], true
}

// HydratedConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be in one of the following forms:
//   - connection://<type>/<name> or
//   - the UUID of the connection.
func HydratedConnectionByURL(ctx context.Context, db *gorm.DB, k8sClient kubernetes.Interface, namespace, connectionString string) (*models.Connection, error) {
	// Must be in one of the correct forms.
	if _, err := uuid.Parse(connectionString); err != nil && !strings.HasPrefix(connectionString, "connection://") {
		return nil, fmt.Errorf("invalid connection string: %q. Expected connection://<type>/<name> or the connection UUID", connectionString)
	}

	connection, err := FindConnectionByURL(ctx, db, connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (%s): %w", connectionString, err)
	}

	if connection == nil {
		return nil, nil
	}

	return HydrateConnection(ctx, k8sClient, db, connection, namespace)
}

func IsValidConnectionURL(connectionString string) bool {
	if _, err := uuid.Parse(connectionString); err == nil {
		return true
	}
	_, _, found := extractConnectionNameType(connectionString)
	return found
}

// FindConnectionByURL retrieves a connection from the given connection string.
// The connection string is expected to be of the form: connection://<type>/<name>
func FindConnectionByURL(ctx context.Context, db *gorm.DB, connectionString string) (*models.Connection, error) {
	if _, err := uuid.Parse(connectionString); err == nil {
		var connection models.Connection
		if err := db.Where("id = ?", connectionString).First(&connection).Error; err != nil {
			return nil, err
		}
		return &connection, nil
	}

	name, connectionType, found := extractConnectionNameType(connectionString)
	if !found {
		return nil, nil
	}

	connection, err := FindConnection(ctx, db, connectionType, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find connection (type=%s, name=%s): %w", connectionType, name, err)
	}

	return connection, nil
}

// FindConnection returns the connection with the given type and name
func FindConnection(ctx context.Context, db *gorm.DB, connectionType, name string) (*models.Connection, error) {
	var connection models.Connection

	err := db.Where("type = ? AND name = ?", connectionType, name).First(&connection).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &connection, nil
}

func GetConnection(ctx context.Context, client kubernetes.Interface, db *gorm.DB, connectionType string, name string, namespace string) (*models.Connection, error) {
	connection, err := FindConnection(ctx, db, connectionType, name)
	if err != nil {
		return nil, err
	}

	if connection == nil {
		return nil, ErrNotFound
	}

	return HydrateConnection(ctx, client, db, connection, namespace)
}

// Create a cache with a default expiration time of 5 minutes, and which
// purges expired items every 10 minutes
// var connectionCache = cache.New(5*time.Minute, 10*time.Minute)
func HydrateConnection(ctx context.Context, client kubernetes.Interface, db *gorm.DB, connection *models.Connection, namespace string) (*models.Connection, error) {
	var err error
	if connection.Username, err = GetEnvStringFromCache(client, connection.Username, namespace); err != nil {
		return nil, err
	}

	if connection.Password, err = GetEnvStringFromCache(client, connection.Password, namespace); err != nil {
		return nil, err
	}

	if connection.Certificate, err = GetEnvStringFromCache(client, connection.Certificate, namespace); err != nil {
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
		"namespace": namespace,
		"username":  connection.Username,
		"password":  connection.Password,
		"domain":    domain,
	}
	templater := template.StructTemplater{
		Values: data,
		// access go values in template requires prefix everything with .
		// to support $(username) instead of $(.username) we add a function for each var
		ValueFunctions: true,
		DelimSets: []template.Delims{
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
