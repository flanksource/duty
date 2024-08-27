package context

import (
	gocontext "context"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metaTypes "k8s.io/apimachinery/pkg/types"
)

type Poolable interface {
	Pool() *pgxpool.Pool
}

type Gormable interface {
	DB() *gorm.DB
}

func Objects(k gocontext.Context) []any {
	objects := k.Value("object")
	switch v := objects.(type) {
	case []any:
		return v
	}
	return nil
}

func objectToMeta(v metav1.Object) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   v.GetNamespace(),
		Name:        v.GetName(),
		Labels:      v.GetLabels(),
		Annotations: v.GetAnnotations(),
		UID:         v.GetUID(),
	}
}

func unstructuredMeta(v unstructured.Unstructured) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:   v.GetNamespace(),
		Name:        v.GetName(),
		Labels:      v.GetLabels(),
		Annotations: v.GetAnnotations(),
		UID:         v.GetUID(),
	}
}

type PKAccessor interface {
	PK() string
}

type NameAccessor interface {
	GetName() string
}

type NamespaceAccess interface {
	GetNamespace() string
}

type LabelsAccessor interface {
	GetLabels() map[string]string
}

type AnnotationsAccessor interface {
	GetAnnotations() map[string]string
}

type ContextAccessor interface {
	Context() map[string]any
}

type ContextAccessor2 interface {
	GetContext() map[string]any
}

func getObjectMeta(o any) metav1.ObjectMeta {
	switch v := o.(type) {
	case metav1.ObjectMeta:
		return v
	case metav1.Object:
		return objectToMeta(v)
	case metav1.ObjectMetaAccessor:
		return objectToMeta(v.GetObjectMeta())
	case unstructured.Unstructured:
		return unstructuredMeta(v)
	}

	out := metav1.ObjectMeta{}

	switch v := o.(type) {
	case models.DBTable:
		if id, err := uuid.Parse(v.PK()); err == nil {
			out.UID = metaTypes.UID(id.String())
		}
	}

	switch v := o.(type) {
	case NameAccessor:
		out.Name = v.GetName()
	}

	switch v := o.(type) {
	case LabelsAccessor:
		out.Labels = v.GetLabels()
	}

	switch v := o.(type) {
	case NamespaceAccess:
		out.Namespace = v.GetNamespace()
	}

	switch v := o.(type) {
	case AnnotationsAccessor:
		out.Annotations = v.GetAnnotations()
	}

	return out
}
