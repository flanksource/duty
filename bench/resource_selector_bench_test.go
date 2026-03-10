package bench_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

const resourceSelectorBenchSize = 200_000

func BenchmarkResourceSelectorConfigs(b *testing.B) {
	resetPG(b, false)
	if err := seedResourceSelectorConfigItems(testCtx, resourceSelectorBenchSize); err != nil {
		b.Fatalf("failed to setup configs: %v", err)
	}

	var names []string
	if err := testCtx.DB().Table("config_items").
		Where("deleted_at IS NULL").
		Order("name").
		Distinct("name").
		Pluck("name", &names).Error; err != nil {
		b.Fatalf("failed to fetch distinct names: %v", err)
	}

	var configTypes []string
	if err := testCtx.DB().Table("config_items").
		Where("deleted_at IS NULL").
		Order("type").
		Distinct("type").
		Pluck("type", &configTypes).Error; err != nil {
		b.Fatalf("failed to fetch distinct types: %v", err)
	}

	tagSelectors := make([]string, len(sampleTags))
	for i, tag := range sampleTags {
		for k, v := range tag {
			tagSelectors[i] = fmt.Sprintf("%s=%s", k, v)
			break
		}
	}

	b.Run("name", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache: "no-cache",
				Name:  names[i%len(names)],
			}
			if _, err := query.FindConfigIDsByResourceSelector(testCtx, -1, selector); err != nil {
				b.Fatalf("query failed: %v", err)
			}
		}
	})

	b.Run("name_and_type", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache: "no-cache",
				Name:  names[i%len(names)],
				Types: types.Items{configTypes[i%len(configTypes)]},
			}
			if _, err := query.FindConfigIDsByResourceSelector(testCtx, -1, selector); err != nil {
				b.Fatalf("query failed: %v", err)
			}
		}
	})

	b.Run("tags", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache:       "no-cache",
				TagSelector: tagSelectors[i%len(tagSelectors)],
			}
			if _, err := query.FindConfigIDsByResourceSelector(testCtx, -1, selector); err != nil {
				b.Fatalf("query failed: %v", err)
			}
		}
	})
}

// BenchmarkResourceSelectorQueryBuild measures only the Go overhead of
// converting a ResourceSelector into a SQL string.
// No database rows are read; no data population is needed.
func BenchmarkResourceSelectorQueryBuild(b *testing.B) {
	names := []string{
		"coredns", "workload-low", "local-path-provisioner",
		"kubeadm-config", "cert-manager", "nginx-ingress",
		"kube-proxy", "etcd-main", "flannel-cni", "metrics-server",
	}

	configTypes := []string{
		"Kubernetes::Pod", "Kubernetes::Deployment",
		"Kubernetes::Node", "Kubernetes::ReplicaSet",
		"Kubernetes::Namespace",
	}

	tagSelectors := make([]string, len(sampleTags))
	for i, tag := range sampleTags {
		for k, v := range tag {
			tagSelectors[i] = fmt.Sprintf("%s=%s", k, v)
			break
		}
	}

	b.Run("name", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache: "no-cache",
				Name:  names[i%len(names)],
			}
			selector = selector.Canonical()
			q := testCtx.DB().Select("id").Table("config_items")
			q, err := query.SetResourceSelectorClause(testCtx, selector, q, "config_items")
			if err != nil {
				b.Fatalf("query build failed: %v", err)
			}
			_ = q.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Find(&[]uuid.UUID{})
			})
		}
	})

	b.Run("name_and_type", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache: "no-cache",
				Name:  names[i%len(names)],
				Types: types.Items{configTypes[i%len(configTypes)]},
			}
			selector = selector.Canonical()
			q := testCtx.DB().Select("id").Table("config_items")
			q, err := query.SetResourceSelectorClause(testCtx, selector, q, "config_items")
			if err != nil {
				b.Fatalf("query build failed: %v", err)
			}
			_ = q.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Find(&[]uuid.UUID{})
			})
		}
	})

	b.Run("tags", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			selector := types.ResourceSelector{
				Cache:       "no-cache",
				TagSelector: tagSelectors[i%len(tagSelectors)],
			}
			selector = selector.Canonical()
			q := testCtx.DB().Select("id").Table("config_items")
			q, err := query.SetResourceSelectorClause(testCtx, selector, q, "config_items")
			if err != nil {
				b.Fatalf("query build failed: %v", err)
			}
			_ = q.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Find(&[]uuid.UUID{})
			})
		}
	})
}
