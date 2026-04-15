// ABOUTME: Tests for upstream push data transformations.
// ABOUTME: Covers AddAgentConfig and related PushData methods.
package upstream

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
)

func TestAddAgentConfig(t *testing.T) {
	agentID := uuid.New()
	agent := models.Agent{
		ID:   agentID,
		Name: "test-agent",
	}

	now := time.Now()

	pushData := &PushData{
		ConfigItems: []models.ConfigItem{
			{
				ID:   models.LocalAgentConfigID,
				Type: lo.ToPtr("MissionControl::ShouldBeFiltered"),
			},
			{
				ID:   uuid.New(),
				Type: lo.ToPtr("MissionControl::Agent"),
			},
		},
		ConfigChanges: []models.ConfigChange{
			{ConfigID: models.LocalAgentConfigID.String()},
		},
		ConfigScrapers: []models.ConfigScraper{
			{ID: uuid.Nil},
			{ID: uuid.New()},
		},
		ConfigItemsLastScrapedTime: []models.ConfigItemLastScrapedTime{
			{ConfigID: models.LocalAgentConfigID, LastScrapedTime: &now},
		},
	}

	pushData.AddAgentConfig(agent)

	// Config item with uuid.Nil ID should be filtered out
	if len(pushData.ConfigItems) != 1 {
		t.Fatalf("expected 1 config item, got %d", len(pushData.ConfigItems))
	}

	// MissionControl::Agent item should have ID set to agent ID
	ci := pushData.ConfigItems[0]
	if ci.ID != agentID {
		t.Errorf("expected agent config item ID to be %s, got %s", agentID, ci.ID)
	}
	if lo.FromPtr(ci.Name) != "test-agent" {
		t.Errorf("expected agent config item name to be 'test-agent', got %s", lo.FromPtr(ci.Name))
	}
	if lo.FromPtr(ci.ScraperID) != uuid.Nil.String() {
		t.Errorf("expected scraper_id to be nil UUID, got %s", lo.FromPtr(ci.ScraperID))
	}

	// Config changes with uuid.Nil config_id should be remapped
	if pushData.ConfigChanges[0].ConfigID != agentID.String() {
		t.Errorf("expected config change config_id to be %s, got %s", agentID, pushData.ConfigChanges[0].ConfigID)
	}

	// System scraper should be filtered out
	if len(pushData.ConfigScrapers) != 1 {
		t.Fatalf("expected 1 config scraper, got %d", len(pushData.ConfigScrapers))
	}
	if pushData.ConfigScrapers[0].ID == uuid.Nil {
		t.Error("system scraper should have been filtered out")
	}

	// Last scraped time with uuid.Nil config_id should be remapped to agent ID
	if len(pushData.ConfigItemsLastScrapedTime) != 1 {
		t.Fatalf("expected 1 last scraped time entry, got %d", len(pushData.ConfigItemsLastScrapedTime))
	}
	if pushData.ConfigItemsLastScrapedTime[0].ConfigID != agentID {
		t.Errorf("expected last scraped time config_id to be %s, got %s", agentID, pushData.ConfigItemsLastScrapedTime[0].ConfigID)
	}
}
