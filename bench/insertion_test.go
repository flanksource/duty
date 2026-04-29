package bench_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	benchAliasUserNamePrefix   = "bench-alias-user-"
	benchAliasScraperName      = "bench-alias-trigger-scraper"
	benchAliasConfigType       = "Bench::AliasTrigger"
	benchPropertyConfigType    = "Bench::Properties"
	benchPropertyUpdateType    = "Bench::PropertiesUpdate"
	benchPropertyPayloadLength = 16 * 1024
	benchPropertyCount         = 6
)

func BenchmarkInsertionForRowsWithAliases(b *testing.B) {
	b.Run("external_users.aliases", func(b *testing.B) {
		resetPG(b, false)
		cleanupExternalUserBenchRows(b)
		scraperID := ensureBenchScraper(b)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			alias := fmt.Sprintf("bench-user-alias-%d", i)
			user := models.ExternalUser{
				ID:        uuid.New(),
				Name:      fmt.Sprintf("%s%d", benchAliasUserNamePrefix, i),
				ScraperID: scraperID,
				Aliases:   pq.StringArray{alias, alias + "-secondary"},
			}

			if err := testCtx.DB().Create(&user).Error; err != nil {
				b.Fatalf("failed to insert external_user #%d: %v", i, err)
			}
		}
		b.StopTimer()
		cleanupExternalUserBenchRows(b)
	})

	b.Run("config_items.external_id", func(b *testing.B) {
		resetPG(b, false)
		cleanupConfigItemBenchRows(b)
		configType := benchAliasConfigType

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			externalID := fmt.Sprintf("bench-config-external-id-%d", i)
			item := models.ConfigItem{
				ID:          uuid.New(),
				ConfigClass: models.ConfigClassNode,
				Type:        &configType,
				ExternalID:  pq.StringArray{externalID, externalID + "-secondary"},
			}

			if err := testCtx.DB().Create(&item).Error; err != nil {
				b.Fatalf("failed to insert config_item #%d: %v", i, err)
			}
		}
		b.StopTimer()
		cleanupConfigItemBenchRows(b)
	})
}

func BenchmarkInsertionOfConfigsWithProperties(b *testing.B) {
	resetPG(b, false)
	cleanupConfigItemBenchRows(b)
	configType := benchPropertyConfigType
	propertyPayload := strings.Repeat("x", benchPropertyPayloadLength)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-config-with-properties-%d", i)
		properties := buildBenchProperties(i, propertyPayload)
		item := models.ConfigItem{
			ID:          uuid.New(),
			ConfigClass: models.ConfigClassNode,
			Type:        &configType,
			Name:        &name,
			Properties:  &properties,
		}

		if err := testCtx.DB().Create(&item).Error; err != nil {
			b.Fatalf("failed to insert config_item with properties #%d: %v", i, err)
		}
	}
	b.StopTimer()
	cleanupConfigItemBenchRows(b)
}

func BenchmarkUpdateOfConfigsWithProperties(b *testing.B) {
	resetPG(b, false)
	cleanupConfigItemBenchRows(b)
	configType := benchPropertyUpdateType
	insertPayload := strings.Repeat("x", benchPropertyPayloadLength)
	updatePayload := strings.Repeat("y", benchPropertyPayloadLength)
	configIDs := make([]uuid.UUID, b.N)

	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-config-update-properties-%d", i)
		properties := buildBenchProperties(i, insertPayload)
		item := models.ConfigItem{
			ID:          uuid.New(),
			ConfigClass: models.ConfigClassNode,
			Type:        &configType,
			Name:        &name,
			Properties:  &properties,
		}

		if err := testCtx.DB().Create(&item).Error; err != nil {
			b.Fatalf("failed to seed config_item with properties #%d: %v", i, err)
		}
		configIDs[i] = item.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-config-updated-properties-%d", i)
		description := fmt.Sprintf("updated config item with properties %d", i)
		properties := buildBenchProperties(i+b.N, updatePayload)

		if err := testCtx.DB().Model(&models.ConfigItem{}).
			Where("id = ?", configIDs[i]).
			Updates(map[string]any{
				"name":        name,
				"description": description,
				"properties":  properties,
			}).Error; err != nil {
			b.Fatalf("failed to update config_item with properties #%d: %v", i, err)
		}
	}
	b.StopTimer()
	cleanupConfigItemBenchRows(b)
}

func buildBenchProperties(seed int, payload string) types.Properties {
	properties := make(types.Properties, 0, benchPropertyCount)
	for i := 0; i < benchPropertyCount; i++ {
		properties = append(properties, &types.Property{
			Name:    fmt.Sprintf("bench-property-%d", i),
			Label:   fmt.Sprintf("Bench Property %d", i),
			Type:    "text",
			Text:    fmt.Sprintf("%s-%d-%d", payload, seed, i),
			Tooltip: fmt.Sprintf("bench property tooltip %d", i),
			Order:   i,
		})
	}
	return properties
}

func cleanupExternalUserBenchRows(b *testing.B) {
	if err := testCtx.DB().Exec("DELETE FROM external_users WHERE name LIKE ?", benchAliasUserNamePrefix+"%").Error; err != nil {
		b.Fatalf("failed to cleanup external_users bench rows: %v", err)
	}
}

func cleanupConfigItemBenchRows(b *testing.B) {
	if err := testCtx.DB().Exec("DELETE FROM config_items WHERE type IN ?", []string{benchAliasConfigType, benchPropertyConfigType, benchPropertyUpdateType}).Error; err != nil {
		b.Fatalf("failed to cleanup config_items bench rows: %v", err)
	}
}

func ensureBenchScraper(b *testing.B) uuid.UUID {
	var scraperIDString string
	if err := testCtx.DB().
		Raw("SELECT id FROM config_scrapers WHERE name = ? AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1", benchAliasScraperName).
		Scan(&scraperIDString).Error; err != nil {
		b.Fatalf("failed to query bench scraper: %v", err)
	}

	if scraperIDString != "" {
		scraperID, err := uuid.Parse(scraperIDString)
		if err != nil {
			b.Fatalf("failed to parse bench scraper id %q: %v", scraperIDString, err)
		}
		return scraperID
	}

	agentID := ensureBenchAgent(b)
	scraperID := uuid.New()
	if err := testCtx.DB().Exec(
		"INSERT INTO config_scrapers (id, name, spec, source, agent_id) VALUES (?, ?, '{}'::jsonb, 'System', ?)",
		scraperID,
		benchAliasScraperName,
		agentID,
	).Error; err != nil {
		b.Fatalf("failed to create bench scraper: %v", err)
	}

	return scraperID
}

func ensureBenchAgent(b *testing.B) uuid.UUID {
	var agentIDString string
	if err := testCtx.DB().
		Raw("SELECT id FROM agents WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT 1").
		Scan(&agentIDString).Error; err != nil {
		b.Fatalf("failed to query agent: %v", err)
	}

	if agentIDString != "" {
		agentID, err := uuid.Parse(agentIDString)
		if err != nil {
			b.Fatalf("failed to parse agent id %q: %v", agentIDString, err)
		}
		return agentID
	}

	agentID := uuid.New()
	if err := testCtx.DB().Exec(
		"INSERT INTO agents (id, name) VALUES (?, ?)",
		agentID,
		"bench-alias-trigger-agent-"+agentID.String(),
	).Error; err != nil {
		b.Fatalf("failed to create bench agent: %v", err)
	}

	return agentID
}
