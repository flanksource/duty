package duty

import (
	gocontext "context"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrNotFound = context.ErrNotFound
)

// deprecated use the method in the context package directly
func HydratedConnectionByURL(ctx gocontext.Context, db *gorm.DB, k8sClient kubernetes.Interface, namespace, connectionString string) (*models.Connection, error) {
	return context.NewContext(ctx).WithKubernetes(k8sClient).WithDB(db, nil).HydratedConnectionByURL(namespace, connectionString)
}

// deprecated use the method in the context package directly
func IsValidConnectionURL(connectionString string) bool {
	return context.IsValidConnectionURL(connectionString)
}

// deprecated use the method in the context package directly
func FindConnectionByURL(ctx context.Context, db *gorm.DB, connectionString string) (*models.Connection, error) {
	return context.FindConnectionByURL(context.NewContext(ctx).WithDB(db, nil), connectionString)
}

// deprecated use the method in the context package directly
func FindConnection(ctx context.Context, db *gorm.DB, connectionType, name string) (*models.Connection, error) {
	return context.FindConnection(context.NewContext(ctx).WithDB(db, nil), connectionType, name)
}

// deprecated use the method in the context package directly
func GetConnection(ctx context.Context, client kubernetes.Interface, db *gorm.DB, connectionType string, name string, namespace string) (*models.Connection, error) {
	return context.GetConnection(context.NewContext(ctx).WithDB(db, nil).WithKubernetes(client), connectionType, name, namespace)
}

// deprecated use the method in the context package directly
func HydrateConnection(ctx context.Context, client kubernetes.Interface, db *gorm.DB, connection *models.Connection, namespace string) (*models.Connection, error) {
	return context.HydrateConnection(context.NewContext(ctx).WithDB(db, nil).WithKubernetes(client), connection, namespace)
}
