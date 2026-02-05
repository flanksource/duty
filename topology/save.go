package topology

import (
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// Save the component and its children returing the ids that were inserted/updated
func SaveComponent(ctx context.Context, c *models.Component) ([]string, error) {
	var ids []string
	if err := saveComponentsRecursively(ctx, c, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// We keep a list of ids to track all the insert/updated ids
func saveComponentsRecursively(ctx context.Context, c *models.Component, ids *[]string) error {
	if c.ParentId != nil && !strings.Contains(c.Path, c.ParentId.String()) {
		if c.Path == "" {
			c.Path = c.ParentId.String()
		} else {
			c.Path += "." + c.ParentId.String()
		}
	}

	if existing, err := query.ComponentFromCache(ctx, c.ID.String(), true); err == nil {
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
		// We set this to nil so that the conflict clause returns correct ID
		c.ID = uuid.Nil
		// Create new component handling conflicts
		if err := ctx.DB().Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "topology_id"}, {Name: "type"}, {Name: "name"}, {Name: "parent_id"}},
				UpdateAll: true,
			}, clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(c).Error; err != nil {
			return db.ErrorDetails(err)
		}
	}

	if ids != nil {
		*ids = append(*ids, c.ID.String())
	}

	if len(c.Components) > 0 {
		for _, child := range c.Components {
			child.TopologyID = c.TopologyID
			child.ParentId = &c.ID
			err := saveComponentsRecursively(ctx, child, ids)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
