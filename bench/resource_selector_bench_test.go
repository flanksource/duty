package bench_test

import (
	"fmt"
	"testing"

	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
)

const resourceSelectorBenchSize = 500_000

func BenchmarkResourceSelectorConfigs(b *testing.B) {
	resetPG(b, false)
	if _, err := setupConfigsForSize(testCtx, resourceSelectorBenchSize); err != nil {
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
