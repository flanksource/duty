package query

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/jackc/pgx/v5"
)

const DefaultDepth = 5

type TopologyOptions struct {
	ID      string
	Owner   string
	Labels  map[string]string
	AgentID string
	Flatten bool
	Depth   int
	// TODO: Filter status and types in DB Query
	Types  []string
	Status []string

	// when set to true, only the children (except the direct children) are returned.
	// when set to false, the direct children & the parent itself is fetched.
	nonDirectChildrenOnly bool
}

func (opt TopologyOptions) String() string {
	return fmt.Sprintf("%#v", opt)
}

// selectClause returns the columns that should be selected from the topology view.
func (opt TopologyOptions) selectClause() string {
	if !opt.nonDirectChildrenOnly {
		return "*"
	}

	// parents & (incidents, analysis, checks) columns need to fetched to create the topology tree even though they may not be essential to the UI.
	return "name, namespace, id, is_leaf, status, status_reason, icon, summary, topology_type, labels, team_names, type, parent_id, parents, incidents, analysis, checks"
}

func (opt TopologyOptions) componentWhereClause() string {
	s := "WHERE components.deleted_at IS NULL"

	if opt.ID != "" {
		if !opt.nonDirectChildrenOnly {
			s += " AND (components.id = @id OR components.parent_id = @id)"
		} else {
			s += " AND (components.path LIKE @path AND components.id != @id AND components.parent_id != @id)"
		}
	} else {
		if !opt.nonDirectChildrenOnly {
			s += " AND components.parent_id IS NULL"
		} else {
			s += " AND components.parent_id IS NOT NULL"
		}
	}

	if opt.Owner != "" {
		s += " AND (components.owner = @owner)"
	}
	if opt.Labels != nil {
		s += " AND (components.labels @> @labels)"
	}
	if opt.AgentID != "" {
		s += " AND (components.agent_id = @agent_id)"
	}
	return s
}

func (opt TopologyOptions) componentRelationWhereClause() string {
	s := "WHERE component_relationships.deleted_at IS NULL AND parent.deleted_at IS NULL"
	if opt.Owner != "" {
		s += ` AND (parent.owner = @owner)`
	}
	if opt.Labels != nil {
		s += ` AND (parent.labels @> @labels)`
	}

	if opt.ID != "" {
		if !opt.nonDirectChildrenOnly {
			s += " AND (component_relationships.relationship_id = @id OR parent.parent_id = @id)"
		} else {
			s += " AND (component_relationships.relationship_id = @id OR (parent.path LIKE @path AND parent.parent_id != @id))"
		}
	} else {
		if !opt.nonDirectChildrenOnly {
			s += " AND component_relationships.component_id IS NULL"
		} else {
			s += " AND component_relationships.component_id IS NOT NULL"
		}
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
	subQuery := fmt.Sprintf(selectSubQuery, opts.componentWhereClause(), opts.componentRelationWhereClause())
	query := fmt.Sprintf(`
        WITH topology_result AS (
            SELECT %s FROM topology
            WHERE id IN (%s)
        )
        SELECT
            json_build_object(
                'components', json_agg(to_jsonb(topology_result)),
                'types', json_agg(DISTINCT(type)),
                'healthStatuses', json_agg(DISTINCT(status)),
                'tags', (SELECT jsonb_object_agg(key, value)
                        FROM (
                            SELECT label->>'key' as key, array_agg(DISTINCT(label->>'value')) AS value
                            FROM (
                                SELECT row_to_json(jsonb_each_text(labels)) AS label FROM topology_result
                            ) AS labels_flat GROUP BY key
                        ) as t2),
                'teams', (SELECT json_agg(team) FROM topology_result, LATERAL unnest(team_names) AS team)
						)
        FROM
            topology_result
        `, opts.selectClause(), subQuery)

	args := make(map[string]any)
	if opts.ID != "" {
		args["id"] = opts.ID
		args["path"] = strings.ReplaceAll(`%id%`, "id", opts.ID)
	}
	if opts.AgentID != "" {
		args["agent_id"] = opts.AgentID
	}
	if opts.Owner != "" {
		args["owner"] = opts.Owner
	}
	if opts.Labels != nil {
		args["labels"] = opts.Labels
	}

	return query, args
}

// Map of tag keys to the list of available values
type Tags map[string][]string

type TopologyResponse struct {
	Components     models.Components `json:"components"`
	HealthStatuses []string          `json:"healthStatuses"`
	Teams          []string          `json:"teams"`
	Tags           Tags              `json:"tags"`
	Types          []string          `json:"types"`
}

func fetchAllComponents(ctx context.Context, params TopologyOptions) (TopologyResponse, error) {
	// Fetch the children (with all the details)
	// & the rest of the decendents (minimal details) in two separate queries

	var response TopologyResponse
	query, args := generateQuery(params)
	rows, err := ctx.Pool().Query(ctx, query, pgx.NamedArgs(args))
	if err != nil {
		return response, fmt.Errorf("failed to query component & its direct children: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &response); err != nil {
			return response, fmt.Errorf("failed to unmarshal TopologyResponse:%w for %s", err, rows.RawValues()[0])
		}
	}

	params.nonDirectChildrenOnly = true
	query, args = generateQuery(params)
	rows, err = ctx.Pool().Query(ctx, query, pgx.NamedArgs(args))
	if err != nil {
		return response, fmt.Errorf("failed to query rest of the children: %w", err)
	}
	defer rows.Close()

	var nonDirectChildren TopologyResponse
	for rows.Next() {
		if rows.RawValues()[0] == nil {
			continue
		}

		if err := json.Unmarshal(rows.RawValues()[0], &nonDirectChildren); err != nil {
			return response, fmt.Errorf("failed to unmarshal TopologyResponse:%w for %s", err, rows.RawValues()[0])
		}
	}

	if len(nonDirectChildren.Components) > 0 {
		compMap := make(map[string]bool)
		for _, c := range response.Components {
			compMap[c.ID.String()] = true
		}
		for _, c := range nonDirectChildren.Components {
			if _, exists := compMap[c.ID.String()]; !exists {
				response.Components = append(response.Components, c)
			}
		}

		response.HealthStatuses = append(response.HealthStatuses, nonDirectChildren.HealthStatuses...)
		response.Teams = append(response.Teams, nonDirectChildren.Teams...)
		response.Types = append(response.Types, nonDirectChildren.Types...)
		if response.Tags != nil || nonDirectChildren.Tags != nil {
			response.Tags = collections.MergeMap(response.Tags, nonDirectChildren.Tags)
		}
	}

	return response, nil
}

func Topology(ctx context.Context, params TopologyOptions) (*TopologyResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(DefaultQueryTimeout)
		defer cancel()
	}

	response, err := fetchAllComponents(ctx, params)
	if err != nil {
		return nil, err
	}

	response.Components = applyTypeFilter(response.Components, params.Types...)

	if !params.Flatten {
		response.Components = createComponentTree(params, response.Components)
	}

	if params.Depth <= 0 {
		params.Depth = DefaultDepth
	}
	response.Components = applyDepthFilter(response.Components, params.Depth)

	// If ID is present, we do not apply any filters to the root component
	response.Components = applyStatusFilter(response.Components, params.ID != "", params.Status...)

	response = updateMetadata(response)

	// Remove fields from children that aren't required by the UI
	root := response.Components
	if len(root) == 1 {
		for j := range root[0].Components {
			removeComponentFields(root[0].Components[j].Components, params.Depth)
		}
	} else {
		for i := range root {
			removeComponentFields(root[i].Components, params.Depth)
		}
	}
	return &response, nil
}

// applyDepthFilter limits the tree size to the given depth and also
// dereferences pointer cycles by creating new copies of components
// to prevent cyclic errors during json.Marshal
func applyDepthFilter(components []*models.Component, depth int) []*models.Component {
	if depth <= 0 || len(components) == 0 {
		return components
	}
	var newComponents []*models.Component
	if depth == 1 {
		for _, comp := range components {
			compCopy := *comp
			compCopy.Components = nil
			newComponents = append(newComponents, &compCopy)
		}
		return newComponents
	}

	for _, comp := range components {
		compCopy := *comp
		compCopy.Components = applyDepthFilter(compCopy.Components, depth-1)
		newComponents = append(newComponents, &compCopy)
	}
	return newComponents
}

func generateTree(components models.Components, compChildrenMap map[string]models.Components, touchedIDs []string) models.Components {
	var nodes models.Components

	for _, c := range components {
		// If node is marked as procesed we can just
		// return it as is since it's child tree is correct
		if c.NodeProcessed {
			nodes = append(nodes, c)
			continue
		}

		c.NodeProcessed = true
		if children, exists := compChildrenMap[c.ID.String()]; exists {
			var childrenToProcess models.Components
			for _, child := range children {
				// If a child has already been part of the tree that
				// can mean we are recursing for the same node again
				// In this case, we skip the tree generation for that
				// node since it already is part of the call stack
				if !collections.Contains(touchedIDs, child.ID.String()) {
					childrenToProcess = append(childrenToProcess, child)
				}
			}

			// TODO: Update fixtures and check that cyclic component must be added for L1
			//touchedIDs = append(touchedIDs, c.ID.String())
			touchedIDs = []string{}

			c.Components = generateTree(childrenToProcess, compChildrenMap, touchedIDs)
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

	var rootComps models.Components
	for _, c := range components {
		// TODO: Try https://stackoverflow.com/questions/30101603/merging-concatenating-jsonb-columns-in-query
		c.Summary.Incidents = c.Incidents
		c.Summary.Insights = c.Analysis
		c.Summary.Checks = c.Checks

		c.Analysis, c.Incidents, c.Checks = nil, nil, nil

		if c.ParentId != nil {
			if _, exists := compChildrenMap[c.ParentId.String()]; exists {
				compChildrenMap[c.ParentId.String()] = append(compChildrenMap[c.ParentId.String()], c)
			}
		}

		// Keep a track of the root components for the current context
		// If params.ID is present only 1 root component can be there
		// else all components without a parent are root
		if params.ID == c.ID.String() && len(rootComps) == 0 {
			rootComps = append(rootComps, c)
		} else if c.ParentId == nil {
			rootComps = append(rootComps, c)
		}

		for _, parentID := range c.Parents {
			compChildrenMap[parentID] = append(compChildrenMap[parentID], c)
		}
	}

	tree := generateTree(rootComps, compChildrenMap, []string{})
	return tree
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

func updateMetadata(resp TopologyResponse) TopologyResponse {
	// Clean teams
	resp.Teams = collections.DeleteEmptyStrings(resp.Teams)

	resp.Teams = collections.Dedup(resp.Teams)
	resp.HealthStatuses = collections.Dedup(resp.HealthStatuses)
	resp.Types = collections.Dedup(resp.Types)

	sort.Slice(resp.Teams, func(i, j int) bool {
		return resp.Teams[i] < resp.Teams[j]
	})

	sort.Slice(resp.HealthStatuses, func(i, j int) bool {
		return resp.HealthStatuses[i] < resp.HealthStatuses[j]
	})

	sort.Slice(resp.Types, func(i, j int) bool {
		return resp.Types[i] < resp.Types[j]
	})

	return resp
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

func GetComponent(ctx context.Context, id string) (*models.Component, error) {
	var component models.Component
	if err := ctx.DB().Where("id = ?", id).First(&component).Error; err != nil {
		return nil, err
	}

	return &component, nil
}

// removeComponentFields recursively removes some of the fields from components
// and their children and so on.
func removeComponentFields(components models.Components, depth int) {
	if depth == 0 {
		return
	}
	for i := range components {
		c := components[i]

		c.ParentId = nil
		c.Parents = nil
		c.Checks = nil
		c.Incidents = nil
		c.Analysis = nil
		removeComponentFields(c.Components, depth-1)
	}
}
