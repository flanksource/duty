package duty

import (
	"context"
	"encoding/json"
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

func (opt TopologyOptions) String() string {
	return fmt.Sprintf("%#v", opt)
}

func (opt TopologyOptions) componentWhereClause() string {
	s := "WHERE components.deleted_at IS NULL "
	if opt.ID != "" {
		s += `and (starts_with(path,
			(SELECT
				(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
				FROM components where id = @id)
			) or id = @id or path = @id :: text)`
	}
	if opt.Owner != "" {
		s += " AND (components.owner = @owner or id = @id)"
	}
	if opt.Labels != nil {
		s += " AND (components.labels @> @labels"
		if opt.ID != "" {
			s += " or id = @id"
		}
		s += ")"
	}
	return s
}

func (opt TopologyOptions) componentRelationWhereClause() string {
	s := "WHERE component_relationships.deleted_at IS NULL and parent.deleted_at IS NULL"
	if opt.Owner != "" {
		s += " AND (parent.owner = @owner)"
	}
	if opt.Labels != nil {
		s += " AND (parent.labels @> @labels)"
	}
	if opt.ID != "" {
		s += ` and (component_relationships.relationship_id = @id or starts_with(component_relationships.relationship_path, (SELECT
			(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
			FROM components where id = @id)))`
	} else {
		s += ` and (parent.parent_id is null or starts_with(component_relationships.relationship_path, (SELECT
			(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
			FROM components where id = parent.id)))`
	}
	return s
}

func generateQuery(opts TopologyOptions) (string, map[string]any) {
	query := fmt.Sprintf(`
    WITH topology_result as (
        SELECT *, NULL AS relationship_id FROM components %s
        UNION (
            SELECT components.*, relationship_id FROM component_relationships
            INNER JOIN components ON components.id = component_relationships.component_id
            INNER JOIN components AS parent ON component_relationships.relationship_id = parent.id %s
        )
    )
	SELECT json_agg(
        jsonb_set_lax(
            jsonb_set_lax(
                jsonb_set_lax(
                    to_jsonb(topology_result),
                        '{checks}', %s
                ), '{summary,insights}', %s
            ), '{summary,incidents}', %s
        )
    ) :: jsonb FROM topology_result`,
		opts.componentWhereClause(), opts.componentRelationWhereClause(), opts.checksForComponents(),
		opts.configAnalysisSummaryForComponents(), opts.incidentSummaryForComponents())

	args := make(map[string]any)
	if opts.ID != "" {
		args["id"] = opts.ID
	}
	if opts.Owner != "" {
		args["owner"] = opts.Owner
	}
	if opts.Labels != nil {
		args["labels"] = opts.Labels
	}
	return query, args
}

func (opts TopologyOptions) checksForComponents() string {
	return `(
        SELECT json_agg(checks) FROM checks
        LEFT JOIN check_component_relationships ON checks.id = check_component_relationships.check_id
        WHERE check_component_relationships.component_id = topology_result.id AND check_component_relationships.deleted_at IS NULL
        GROUP BY check_component_relationships.component_id
    ) :: jsonb`
}

func (opts TopologyOptions) configAnalysisSummaryForComponents() string {
	return `(SELECT analysis FROM analysis_summary_by_component WHERE id = topology_result.id)`
}

func (p TopologyOptions) incidentSummaryForComponents() string {
	return `(SELECT incidents FROM incident_summary_by_component WHERE id = topology_result.id)`
}

func QueryTopology(dbpool *pgxpool.Pool, params TopologyOptions) ([]*models.Component, error) {
	query, args := generateQuery(params)
	rows, err := dbpool.Query(context.Background(), query, pgx.NamedArgs(args))
	if err != nil {
		return nil, err
	}

	var results models.Components
	for rows.Next() {
		var components models.Components
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &components); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components:%v for %s", err, rows.RawValues()[0])
		}
		results = append(results, components...)
	}

	results = applyTypeFilter(results, params.Types...)
	if !params.Flatten {
		results = createComponentTree(params, results)
	}

	if params.Depth <= 0 {
		params.Depth = DefaultDepth
	}
	results = applyDepthFilter(results, params.Depth)

	results = applyStatusFilter(results, params.Status...)

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

func applyStatusFilter(components []*models.Component, statii ...string) []*models.Component {
	if len(statii) == 0 {
		return components
	}
	var filtered []*models.Component
	for _, component := range components {
		if matchItems(string(component.Status), statii...) {
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
