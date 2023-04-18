package duty

import (
	"context"
	"fmt"
	"strings"

	"github.com/flanksource/duty/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

const DefaultDepth = 5

type TopologyOptions struct {
	ID      string
	Owner   string
	Labels  map[string]string
	Flatten bool
	Depth   int
	Types   []string
	Status  []string
}

func generateQuery(opts TopologyOptions) (string, map[string]any) {
	query := `
SELECT
  id,
  system_template_id,
  external_id,
  parent_id,
  name,
  text,
  topology_type,
  namespace,
  labels,
  hidden,
  silenced,
  status,
  COALESCE(description, ''),
  lifecycle,
  tooltip,
  status_reason,
  schedule,
  icon,
  type,
  owner,
	configs,
  path,
  summary,
  is_leaf,
  created_by,
  created_at,
  updated_at,
  deleted_at,
  log_selectors,
  checks
FROM
  topology`

	args := make(map[string]any)
	if opts.ID != "" {
		query = fmt.Sprintf("%s WHERE id = @id OR parent_id = @id", query)
		args["id"] = opts.ID
	}

	if opts.Owner != "" {
		query = fmt.Sprintf("%s WHERE owner = @owner", query)
		args["owner"] = opts.Owner
	}

	if len(opts.Labels) > 0 {
		query = fmt.Sprintf("%s WHERE labels @> @labels", query)
		args["labels"] = opts.Labels
	}

	return query, args
}

func scanComponent(row pgx.Rows) (*models.Component, error) {
	var c models.Component
	err := row.Scan(
		&c.ID,
		&c.SystemTemplateID,
		&c.ExternalId,
		&c.ParentId,
		&c.Name,
		&c.Text,
		&c.TopologyType,
		&c.Namespace,
		&c.Labels,
		&c.Hidden,
		&c.Silenced,
		&c.Status,
		&c.Description,
		&c.Lifecycle,
		&c.Tooltip,
		&c.StatusReason,
		&c.Schedule,
		&c.Icon,
		&c.Type,
		&c.Owner,
		&c.Configs,
		&c.Path,
		&c.Summary,
		&c.IsLeaf,
		&c.CreatedBy,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
		&c.LogSelectors,
		&c.Checks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan component: %w", err)
	}

	return &c, nil
}

func QueryTopology(ctx context.Context, dbpool *pgxpool.Pool, params TopologyOptions) ([]*models.Component, error) {
	query, args := generateQuery(params)
	rows, err := dbpool.Query(ctx, query, pgx.NamedArgs(args))
	if err != nil {
		return nil, fmt.Errorf("failed to query topology: %w", err)
	}

	var results models.Components
	for rows.Next() {
		component, err := scanComponent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan to component: %w", err)
		}

		results = append(results, component)
	}

	results = applyTypeFilter(results, params.Types...)
	if !params.Flatten {
		results = createComponentTree(params, results)
	}

	if params.Depth <= 0 {
		params.Depth = DefaultDepth
	}
	results = applyDepthFilter(results, params.Depth)

	// If ID is present, we do not apply any filters to the root component
	results = applyStatusFilter(results, params.ID != "", params.Status...)

	return results, nil
}

func applyDepthFilter(components []*models.Component, depth int) []*models.Component {
	if depth <= 0 || len(components) == 0 {
		return components
	}
	if depth == 1 {
		for _, comp := range components {
			comp.Components = nil
		}
		return components
	}

	for _, comp := range components {
		comp.Components = applyDepthFilter(comp.Components, depth-1)
	}
	return components
}

func generateTree(components []*models.Component, compChildrenMap map[string][]*models.Component) []*models.Component {
	var nodes []*models.Component
	for _, c := range components {
		if children, exists := compChildrenMap[c.ID.String()]; exists {
			c.Components = generateTree(children, compChildrenMap)
		}
		c.Summary = c.Summarize()

		c.Status = c.GetStatus()
		nodes = append(nodes, c)
	}
	return nodes
}

func createComponentTree(params TopologyOptions, components models.Components) []*models.Component {
	// ComponentID with its children
	compChildrenMap := make(map[string][]*models.Component)

	for _, c := range components {
		compChildrenMap[c.ID.String()] = []*models.Component{}
	}

	for _, c := range components {
		if c.ParentId != nil {
			if _, exists := compChildrenMap[c.ParentId.String()]; exists {
				compChildrenMap[c.ParentId.String()] = append(compChildrenMap[c.ParentId.String()], c)
			}
		}
		if c.RelationshipID != nil {
			if _, exists := compChildrenMap[c.RelationshipID.String()]; exists {
				compChildrenMap[c.RelationshipID.String()] = append(compChildrenMap[c.RelationshipID.String()], c)
			}
		}
	}

	tree := generateTree(components, compChildrenMap)
	var root models.Components
	for _, c := range tree {
		if c.ParentId == nil || params.ID == c.ID.String() {
			root = append(root, c)
		}
	}
	return root
}

func applyTypeFilter(components []*models.Component, types ...string) []*models.Component {
	if len(types) == 0 {
		return components
	}

	var filtered []*models.Component
	for _, component := range components {
		if matchItems(component.Type, types...) {
			filtered = append(filtered, component)
		}
	}
	return filtered
}

func applyStatusFilter(components []*models.Component, filterRoot bool, statii ...string) []*models.Component {
	if len(statii) == 0 {
		return components
	}
	var filtered []*models.Component
	for _, component := range components {
		if filterRoot || matchItems(string(component.Status), statii...) {
			filtered = append(filtered, component)
		}
		var filteredChildren []*models.Component
		for _, child := range component.Components {
			if matchItems(string(child.Status), statii...) {
				filteredChildren = append(filteredChildren, child)
			}
		}
		component.Components = filteredChildren
	}
	return filtered
}

// matchItems returns true if any of the items in the list match the item
// negative matches are supported by prefixing the item with a !
// * matches everything
func matchItems(item string, items ...string) bool {
	if len(items) == 0 {
		return true
	}

	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			if item == strings.TrimPrefix(i, "!") {
				return false
			}
		}
	}

	for _, i := range items {
		if strings.HasPrefix(i, "!") {
			continue
		}
		if i == "*" || item == i {
			return true
		}
	}
	return false
}

func GetComponent(ctx context.Context, db *gorm.DB, id string) (*models.Component, error) {
	var component models.Component
	if err := db.WithContext(ctx).Where("id = ?", id).First(&component).Error; err != nil {
		return nil, err
	}

	return &component, nil
}
