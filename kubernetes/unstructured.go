package kubernetes

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var diffMatchPatch = diffmatchpatch.New()
var defaulter = runtime.NewScheme()

func hasChanges(from, to *unstructured.Unstructured) bool {
	return diff(from, to) != ""
}

func diff(from, to *unstructured.Unstructured) string {
	_from := from.DeepCopy()
	_to := to.DeepCopy()
	Sanitize(_from, _to)

	_fromYaml := ToYaml(_from)
	_toYaml := ToYaml(_to)
	if _fromYaml == _toYaml {
		return ""
	}
	diffs := diffMatchPatch.DiffMain(_fromYaml, _toYaml, false)
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			return diffMatchPatch.DiffPrettyText(diffs)
		}
	}

	return ""
}

// Sanitize will remove "runtime" fields from objects that woulds otherwise increase the verbosity of diffs
func Sanitize(objects ...*unstructured.Unstructured) {
	for _, unstructuredObj := range objects {
		// unstructuredObj.SetCreationTimestamp(metav1.Time{})
		if unstructuredObj.GetAnnotations() == nil {
			unstructuredObj.SetAnnotations(make(map[string]string))
		}
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "creationTimestamp")
		unstructured.RemoveNestedField(unstructuredObj.Object, "creationTimestamp")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "managedFields")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "ownerReferences")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "generation")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "uid")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "selfLink")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "resourceVersion")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "annotations", "deprecated.daemonset.template.generation")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "annotations", "template-operator-owner-ref")
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "annotations", "deployment.kubernetes.io/revision")
		unstructured.RemoveNestedField(unstructuredObj.Object, "status")
		unstructured.RemoveNestedField(unstructuredObj.Object, "spec", "template", "metadata", "creationTimestamp")
	}
}

func requiresReplacement(obj *unstructured.Unstructured, err error) bool {
	if err == nil {
		return false
	}

	switch obj.GetKind() {
	case "Deployment", "DaemonSet":
		return strings.Contains(err.Error(), "field is immutable")
	case "RoleBinding", "ClusterRoleBinding":
		return strings.Contains(err.Error(), "cannot change roleRef")
	default:
		return false
	}
}

func StripIdentifiers(object *unstructured.Unstructured) *unstructured.Unstructured {
	object.SetResourceVersion("")
	object.SetSelfLink("")
	object.SetUID("")
	object.SetCreationTimestamp(metav1.Time{})
	object.SetGeneration(0)
	return object
}
