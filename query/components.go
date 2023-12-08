package query

import (
	"github.com/flanksource/duty/context"
	"github.com/google/uuid"
)

func FindComponentIDsByNameNamespaceType(ctx context.Context, namespace, name, componentType string) ([]uuid.UUID, error) {
	return lookupIDs(ctx, "components", namespace, name, componentType)
}
