package upstream

import (
	"errors"
	"fmt"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func getAgent(ctx context.Context, name string) (*models.Agent, error) {
	var t models.Agent
	tx := ctx.DB().Where("name = ?", name).First(&t)
	return &t, tx.Error
}

func createAgent(ctx context.Context, name string) (*models.Agent, error) {
	a := models.Agent{Name: name}
	tx := ctx.DB().Create(&a)
	return &a, tx.Error
}

func GetOrCreateAgent(ctx context.Context, name string) (*models.Agent, error) {
	a, err := getAgent(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newAgent, err := createAgent(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("failed to create agent: %w", err)
			}
			return newAgent, nil
		}
		return nil, err
	}

	return a, nil
}

// DeleteOnUpstream deletes the given resources by agent on the upstream.
func DeleteOnUpstream(ctx context.Context, req *PushData) error {
	db := ctx.DB()

	if len(req.Topologies) > 0 {
		if err := db.Delete(req.Topologies).Error; err != nil {
			return fmt.Errorf("error deleting topologies: %w", err)
		}
	}

	if len(req.Canaries) > 0 {
		if err := db.Delete(req.Canaries).Error; err != nil {
			return fmt.Errorf("error deleting canaries: %w", err)
		}
	}

	if len(req.Components) > 0 {
		if err := db.Delete(req.Components).Error; err != nil {
			logger.Errorf("error deleting components: %w", err)
		}
	}

	if len(req.ComponentRelationships) > 0 {
		if err := db.Delete(req.ComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting component_relationships: %w", err)
		}
	}

	if len(req.ConfigScrapers) > 0 {
		if err := db.Delete(req.ConfigScrapers).Error; err != nil {
			return fmt.Errorf("error deleting config scrapers: %w", err)
		}
	}

	if len(req.ConfigItems) > 0 {
		if err := db.Delete(req.ConfigItems).Error; err != nil {
			logger.Errorf("error deleting config items: %w", err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		if err := db.Delete(req.ConfigRelationships).Error; err != nil {
			return fmt.Errorf("error deleting config_relationships: %w", err)
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		if err := db.Delete(req.ConfigComponentRelationships).Error; err != nil {
			return fmt.Errorf("error deleting config_component_relationships: %w", err)
		}
	}

	if len(req.Checks) > 0 {
		if err := db.Delete(req.Checks).Error; err != nil {
			return fmt.Errorf("error deleting checks: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		if err := db.Delete(req.CheckStatuses).Error; err != nil {
			return fmt.Errorf("error deleting check_statuses: %w", err)
		}
	}

	return nil
}

func InsertUpstreamMsg(ctx context.Context, req *PushData) error {
	batchSize := 100
	db := ctx.DB()
	if len(req.Topologies) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Topologies, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting topologies: %w", err)
		}
	}

	if len(req.Canaries) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Canaries, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting canaries: %w", err)
		}
	}

	// components are inserted one by one, instead of in a batch, because of the foreign key constraint with itself.
	for _, c := range req.Components {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Components, batchSize).Error; err != nil {
			logger.Errorf("error upserting component (id=%s): %v", c.ID, err)
		}
	}

	if len(req.ComponentRelationships) > 0 {
		cols := []clause.Column{{Name: "component_id"}, {Name: "relationship_id"}, {Name: "selector_id"}}
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: cols}).CreateInBatches(req.ComponentRelationships, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting component_relationships: %w", err)
		}
	}

	if len(req.ConfigScrapers) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.ConfigScrapers, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting config scrapers: %w", err)
		}
	}

	// config items are inserted one by one, instead of in a batch, because of the foreign key constraint with itself.
	for _, ci := range req.ConfigItems {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(&ci, batchSize).Error; err != nil {
			logger.Errorf("error upserting config item (id=%s): %v", ci.ID, err)
		}
	}

	if len(req.ConfigRelationships) > 0 {
		cols := []clause.Column{{Name: "related_id"}, {Name: "config_id"}, {Name: "selector_id"}}
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: cols}).CreateInBatches(req.ConfigRelationships, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting config_relationships: %w", err)
		}
	}

	if len(req.ConfigComponentRelationships) > 0 {
		cols := []clause.Column{{Name: "component_id"}, {Name: "config_id"}}
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: cols}).CreateInBatches(req.ConfigComponentRelationships, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting config_component_relationships: %w", err)
		}
	}

	if len(req.ConfigChanges) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.ConfigChanges, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting config_changes: %w", err)
		}
	}

	if len(req.ConfigAnalysis) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.ConfigAnalysis, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting config_analysis: %w", err)
		}
	}

	if len(req.Checks) > 0 {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(req.Checks, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting checks: %w", err)
		}
	}

	if len(req.CheckStatuses) > 0 {
		cols := []clause.Column{{Name: "check_id"}, {Name: "time"}}
		if err := db.Clauses(clause.OnConflict{UpdateAll: true, Columns: cols}).CreateInBatches(req.CheckStatuses, batchSize).Error; err != nil {
			return fmt.Errorf("error upserting check_statuses: %w", err)
		}
	}

	return nil
}
