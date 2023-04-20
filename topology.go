package duty

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flanksource/commons/logger"
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
		s += `AND (id = @id OR path LIKE '%@id%')`
	}
	if opt.Owner != "" {
		s += " AND (components.owner = @owner)"
	}
	if opt.Labels != nil {
		s += " AND (components.labels @> @labels)"
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
		s += ` AND (component_relationships.relationship_id = @id OR parent.path LIKE '%@id%')`
	}
	return s
}

func generateQuery(opts TopologyOptions) (string, map[string]any) {
	selectSubQuery := `
        SELECT id FROM components %s
        UNION
        SELECT component_id FROM component_relationships
        INNER JOIN components ON components.id = component_relationships.component_id
        INNER JOIN components AS parent ON component_relationships.relationship_id = parent.id %s
    `
	subq := fmt.Sprintf(selectSubQuery, opts.componentWhereClause(), opts.componentRelationWhereClause())
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

	newQuery := `
        WITH topology_result AS (
            SELECT * FROM topology
            WHERE id IN (%s)
        )
        SELECT
            json_build_object(
                --'components', json_agg(json_build_object('id', id, 'name', name)),
                'components', json_agg(to_jsonb(topology_result)),
                'types', json_agg(DISTINCT(type)),
                'healthStatuses', json_agg(DISTINCT(status)),
                'tags', (SELECT json_agg(json_build_object(key, val))
                        FROM (
                            SELECT k->>'key' as key, array_agg(distinct(k->>'value')) as val
                            FROM (
                                SELECT row_to_json(jsonb_each_text(labels)) as k FROM topology_result
                            ) as t group by 1
                        ) as t2),
                'teams', json_agg(DISTINCT(team_names))
            )
        FROM
            topology_result
    `
	query = fmt.Sprintf(newQuery, subq)
	logger.Infof("QUERY IS %s", query)
	var clauses []string
	args := make(map[string]any)
	if opts.ID != "" {
		clauses = append(clauses, "(id = @id OR path LIKE %@id%)")
		args["id"] = opts.ID
	}
	if opts.Owner != "" {
		args["owner"] = opts.Owner
		clauses = append(clauses, "owner = @owner")
	}
	if opts.Labels != nil {
		args["labels"] = opts.Labels
		clauses = append(clauses, "labels @> @labels")
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

type Tag map[string][]string

type TopRes struct {
	Components     models.Components `json:"components"`
	HealthStatuses []string          `json:"healthStatuses,omitempty"`
	Teams          []string          `json:"teams,omitempty"`
	Tags           []Tag             `json:"tags,omitempty"`
	Types          []string          `json:"types,omitempty"`
}

func QueryTopology(dbpool *pgxpool.Pool, params TopologyOptions) (*TopRes, error) {
	query, args := generateQuery(params)
	rows, err := dbpool.Query(context.Background(), query, pgx.NamedArgs(args))
	if err != nil {
		return nil, err
	}

	var results models.Components
	var topres TopRes
	for rows.Next() {
		var components models.Components
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &topres); err != nil {
			return nil, fmt.Errorf("failed to unmarshal components:%v for %s", err, rows.RawValues()[0])
		}
		logger.Infof("TOP RES is %s", topres)
		results = append(results, components...)
	}

	topres.Components = applyTypeFilter(topres.Components, params.Types...)
	//params.Flatten = true
	if !params.Flatten {
		topres.Components = createComponentTree(params, topres.Components)
	}

	if params.Depth <= 0 {
		params.Depth = DefaultDepth
	}
	topres.Components = applyDepthFilter(topres.Components, params.Depth)

	// If ID is present, we do not apply any filters to the root component
	topres.Components = applyStatusFilter(topres.Components, params.ID != "", params.Status...)

	return &topres, nil
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

func generateTree(components models.Components, compChildrenMap map[string]models.Components) []*models.Component {
	var nodes models.Components
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
	compChildrenMap := make(map[string]models.Components)

	for _, c := range components {
		compChildrenMap[c.ID.String()] = models.Components{}
	}

	for _, c := range components {
		if c.ParentId != nil {
			if _, exists := compChildrenMap[c.ParentId.String()]; exists {
				if c.ID.String() == c.ParentId.String() {
					logger.Infof("IDS ARE EQUAL %s", c.ID)
				}
				compChildrenMap[c.ParentId.String()] = append(compChildrenMap[c.ParentId.String()], c)
			}
		}
		logger.Infof("Outside Comp ID: %s || Rel ID got %s", c.ID, c.RelationshipID)
		if c.RelationshipID != nil {
			if _, exists := compChildrenMap[c.RelationshipID.String()]; exists {
				logger.Infof("Comp ID: %s || Rel ID got %s", c.ID, c.RelationshipID)
				if c.ID.String() != c.RelationshipID.String() {
					logger.Infof("REL IDS ARE EQUAL %s", c.ID)
					compChildrenMap[c.RelationshipID.String()] = append(compChildrenMap[c.RelationshipID.String()], c)
				}
			}
		}
		//if len(c.Parents) > 0 {
		//compChildrenMap[c.Parents[0]] = append(compChildrenMap[c.Parents[0]], c)
		//}
		for _, parentID := range c.Parents {
			compChildrenMap[parentID] = append(compChildrenMap[parentID], c)
		}
	}

	logger.Infof("COMP CHILD MAP: %s", compChildrenMap)
	for k, v := range compChildrenMap {
		logger.Infof("For parent %s", k)
		for _, vv := range v {
			logger.Infof("Children are %s", vv.ID)
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
