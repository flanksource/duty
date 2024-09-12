package topology

import (
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"gorm.io/gorm/clause"
)

func SaveComponent(ctx context.Context, c *models.Component) error {
	if c.ParentId != nil && !strings.Contains(c.Path, c.ParentId.String()) {
		if c.Path == "" {
			c.Path = c.ParentId.String()
		} else {
			c.Path += "." + c.ParentId.String()
		}
	}

	if existing, err := query.ComponentFromCache(ctx, c.ID.String()); err == nil {
		// Update component if it exists
		if err := ctx.DB().UpdateColumns(c).Error; err != nil {
			return db.ErrorDetails(err)
		}

		// Unset deleted_at if it was non nil
		if existing.DeletedAt != nil && c.DeletedAt == nil {
			if err := ctx.DB().Update("deleted_at", nil).Error; err != nil {
				return db.ErrorDetails(err)
			}
		}
	} else {
		// Create new component handling conflicts
		if err := ctx.DB().Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "topology_id"}, {Name: "type"}, {Name: "name"}, {Name: "parent_id"}},
				UpdateAll: true,
			}).Create(c).Error; err != nil {
			return db.ErrorDetails(err)
		}
	}

	if len(c.Components) > 0 {
		for _, child := range c.Components {
			child.TopologyID = c.TopologyID
			child.ParentId = &c.ID
			if err := SaveComponent(ctx, child); err != nil {
				return err
			}
		}
	}
	return nil
}
