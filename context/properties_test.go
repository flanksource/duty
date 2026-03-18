package context

import (
	gocontext "context"
	"testing"

	"github.com/flanksource/commons/properties"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPropertiesAnnotationOverride(t *testing.T) {
	ctx := NewContext(gocontext.TODO()).WithObject(metav1.ObjectMeta{
		Name:      "test-scraper",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
			"other-annotation":                          "ignored",
		},
	})

	props := ctx.Properties()
	if !props.On(false, "scraper.log.relationships") {
		t.Error("expected annotation to set property to true")
	}
	if props.String("other-annotation", "") != "" {
		t.Error("non mission-control/ annotations should not appear in properties")
	}
}

func TestPropertiesAnnotationDoesNotHideGlobal(t *testing.T) {
	properties.Global.Set("some.global.prop", "globalvalue")
	defer properties.Global.Set("some.global.prop", "")

	ctx := NewContext(gocontext.TODO()).WithObject(metav1.ObjectMeta{
		Name:      "test-scraper-2",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
		},
	})

	props := ctx.Properties()
	if props.String("some.global.prop", "") != "globalvalue" {
		t.Error("annotation merge should not hide other global properties")
	}
	if !props.On(false, "scraper.log.relationships") {
		t.Error("annotation property should be accessible")
	}
}

func TestPropertiesGlobalTakesPrecedenceOverAnnotation(t *testing.T) {
	properties.Global.Set("scraper.log.relationships", "false")
	defer properties.Global.Set("scraper.log.relationships", "")

	ctx := NewContext(gocontext.TODO()).WithObject(metav1.ObjectMeta{
		Name:      "test-scraper-3",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
		},
	})

	props := ctx.Properties()
	if props.On(false, "scraper.log.relationships") {
		t.Error("properties.Global should take precedence over annotations")
	}
}

func TestPropertiesNoAnnotations(t *testing.T) {
	properties.Global.Set("scraper.log.missing", "true")
	defer properties.Global.Set("scraper.log.missing", "")

	ctx := NewContext(gocontext.TODO())
	props := ctx.Properties()
	if !props.On(false, "scraper.log.missing") {
		t.Error("expected global property to be true")
	}
}

func TestPropertiesCachedPerObject(t *testing.T) {
	ctx := NewContext(gocontext.TODO()).WithObject(metav1.ObjectMeta{
		Name:      "cached-scraper",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
		},
	})

	props1 := ctx.Properties()
	props2 := ctx.Properties()
	if !props1.On(false, "scraper.log.relationships") || !props2.On(false, "scraper.log.relationships") {
		t.Error("cached properties should preserve annotation values")
	}
}

func TestHierarchicalPropertiesChildOverridesParent(t *testing.T) {
	parent := metav1.ObjectMeta{
		Name:      "parent",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.batch.size": "100",
		},
	}
	child := metav1.ObjectMeta{
		Name:      "child",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.batch.size": "500",
		},
	}

	ctx := NewContext(gocontext.TODO()).WithObject(parent, child)
	props := ctx.Properties()
	if got := props.Int("scraper.batch.size", 0); got != 500 {
		t.Errorf("expected child value 500, got %d", got)
	}
}

func TestHierarchicalPropertiesParentFallback(t *testing.T) {
	parent := metav1.ObjectMeta{
		Name:      "parent",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.timeout": "30s",
		},
	}
	child := metav1.ObjectMeta{
		Name:        "child",
		Namespace:   "default",
		Annotations: map[string]string{},
	}

	ctx := NewContext(gocontext.TODO()).WithObject(parent, child)
	props := ctx.Properties()
	if got := props.String("scraper.timeout", ""); got != "30s" {
		t.Errorf("expected parent value '30s', got %q", got)
	}
}

func TestHierarchicalPropertiesGlobalFallback(t *testing.T) {
	ctx := NewContext(gocontext.TODO()).WithObject(metav1.ObjectMeta{
		Name:        "obj",
		Namespace:   "default",
		Annotations: map[string]string{},
	})

	props := ctx.Properties()
	props.global = Properties{"global.only.key": "global-value"}

	if got := props.String("global.only.key", ""); got != "global-value" {
		t.Errorf("expected global value 'global-value', got %q", got)
	}
}

func TestHierarchicalPropertiesGlobalWins(t *testing.T) {
	properties.Global.Set("scraper.log.relationships", "false")
	defer properties.Global.Set("scraper.log.relationships", "")

	parent := metav1.ObjectMeta{
		Name:      "parent",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
		},
	}
	child := metav1.ObjectMeta{
		Name:      "child",
		Namespace: "default",
		Annotations: map[string]string{
			"mission-control/scraper.log.relationships": "true",
		},
	}

	ctx := NewContext(gocontext.TODO()).WithObject(parent, child)
	props := ctx.Properties()
	if props.On(false, "scraper.log.relationships") {
		t.Error("CLI/env global should override local annotations at all levels")
	}
}
