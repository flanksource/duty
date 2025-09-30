package query

import (
	gocontext "context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flanksource/commons/collections"
	gocache "github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

func (opt TopologyOptions) String() string {
	return fmt.Sprintf("%#v", opt)
}

func FlushTopologyCache() {
	topologyCache.Flush()
}

// selectClause returns the columns that should be selected from the topology view.
func (opt TopologyOptions) selectClause() string {
	if !opt.nonDirectChildrenOnly {
		return "*"
	}

	// parents & (incidents, analysis, checks) columns need to fetched to create the topology tree even though they may not be essential to the UI.
	return "name, namespace, id, is_leaf, properties, status, status_expr, health, health_expr, status_reason, icon, summary, topology_type, labels, team_names, type, parent_id, parents, incidents, analysis, checks"
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

func generateQuery(opts TopologyOptions) (string, []any) {
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

	var args []any
	if opts.ID != "" {
		args = append(args, sql.Named("id", opts.ID))
		args = append(args, sql.Named("path", strings.ReplaceAll(`%id%`, "id", opts.ID)))
	}
	if opts.AgentID != "" {
		args = append(args, sql.Named("agent_id", opts.AgentID))
	}
	if opts.Owner != "" {
		args = append(args, sql.Named("owner", opts.Owner))
	}
	if opts.Labels != nil {
		args = append(args, sql.Named("labels", opts.Labels))
	}

	return query, args
}

var topologyCache = gocache.New(1*time.Hour, 15*time.Minute)

func fetchAllComponents(ctx context.Context, params TopologyOptions) (TopologyResponse, error) {
	// Fetch the children (with all the details)
	// & the rest of the descendants (minimal details) in two separate queries
	var response TopologyResponse
	var cacheKey string

	ctx.GetSpan().SetAttributes(
		attribute.String("query.id", params.ID),
		attribute.String("query.agent_id", params.AgentID),
		attribute.Int("query.depth", params.Depth),
	)

	if !params.NoCache {
		cacheKey = params.CacheKey()

		// Note: When accessed via mission-control (i.e. when User is set),
		// we need to cache per user due to permission differences.
		if user := ctx.User(); user != nil {
			cacheKey = fmt.Sprintf("%s-%s", cacheKey, user.ID.String())
		}

		if cached, ok := topologyCache.Get(cacheKey); ok {
			ctx.GetSpan().SetAttributes(
				attribute.Bool("cache.hit", true),
			)
			ctx.Logger.V(3).Infof("cache hit: %v", params.String())
			err := json.Unmarshal(cached.([]byte), &response)
			return response, err
		}
	}
	ctx.GetSpan().SetAttributes(
		attribute.Bool("cache.hit", false),
	)
	query, args := generateQuery(params)
	rows, err := ctx.DB().Raw(query, args...).Rows()
	if err != nil {
		return response, fmt.Errorf("failed to query component & its direct children: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jsonData []byte
		if err := rows.Scan(&jsonData); err != nil {
			return response, fmt.Errorf("failed to scan row: %w", err)
		}

		if jsonData == nil {
			continue
		}

		if err := json.Unmarshal(jsonData, &response); err != nil {
			return response, fmt.Errorf("failed to unmarshal TopologyResponse for %s: %w", jsonData, err)
		}
	}

	params.nonDirectChildrenOnly = true
	query, args = generateQuery(params)
	rows, err = ctx.DB().Raw(query, args...).Rows()
	if err != nil {
		return response, fmt.Errorf("failed to query rest of the children: %w", err)
	}
	defer rows.Close()

	var nonDirectChildren TopologyResponse
	for rows.Next() {
		var jsonData []byte
		if err := rows.Scan(&jsonData); err != nil {
			return response, fmt.Errorf("failed to scan row: %w", err)
		}

		if jsonData == nil {
			continue
		}

		if err := json.Unmarshal(jsonData, &nonDirectChildren); err != nil {
			return response, fmt.Errorf("failed to unmarshal TopologyResponse for %s: %w", jsonData, err)
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

	if !params.NoCache {
		data, _ := json.Marshal(response)
		topologyCache.Set(cacheKey, data, ctx.Properties().Duration("topology.cache.age", time.Minute*5))
	}
	return response, nil
}

func Topology(ctx context.Context, params TopologyOptions) (*TopologyResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel gocontext.CancelFunc
		ctx, cancel = ctx.WithTimeout(ctx.Properties().Duration("topology.query.timeout", DefaultQueryTimeout))
		defer cancel()
	}
	ctx, span := ctx.StartSpan("TopologyQuery")
	defer span.End()

	response, err := fetchAllComponents(ctx, params)
	if err != nil {
		return nil, err
	}

	response.Components = applyTypeFilter(response.Components, params.Types...)

	if !params.Flatten {
		response.Components, err = createComponentTree(params, response.Components)
		if err != nil {
			return nil, err
		}
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
			removeComponentFields(root[0].Components[j].Components)
		}

		if params.SortBy != "" {
			SortComponentsByField(root[0].Components, params.SortBy, params.SortOrder != "desc")
		}
	} else {
		for i := range root {
			removeComponentFields(root[i].Components)
		}

		if params.SortBy != "" {
			SortComponentsByField(root, params.SortBy, params.SortOrder != "desc")
		}
	}
	return &response, nil
}

func isZeroVal[T string | *int64](v T) bool {
	var z T
	return v == z
}

func SortComponentsByField(c models.Components, sortBy TopologyQuerySortBy, asc bool) {
	switch {
	case sortBy == TopologyQuerySortByName:
		sort.Slice(c, func(i, j int) bool {
			if !asc {
				i, j = j, i
			}
			return c[i].Name < c[j].Name
		})

	case strings.HasPrefix(string(sortBy), string(TopologyQuerySortByField)):
		field := strings.TrimPrefix(string(sortBy), string(TopologyQuerySortByField))
		isTextProperty := lo.Reduce(c, func(val bool, comp *models.Component, _ int) bool {
			return val && comp.Properties.Find(field).Text != ""
		}, true)

		sort.Slice(c, func(i, j int) bool {
			if !asc {
				i, j = j, i
			}
			propI := c[i].Properties.Find(field)
			propJ := c[j].Properties.Find(field)
			if propI == nil || propJ == nil {
				return false
			}

			// Zero values should always be pushed to the end
			// Since order is dependent on `asc` we negate it
			if isTextProperty {
				if isZeroVal(propI.Text) {
					return !asc
				}
				if isZeroVal(propJ.Text) {
					return asc
				}

				return propI.Text < propJ.Text
			} else {
				if isZeroVal(propI.Value) {
					return !asc
				}
				if isZeroVal(propJ.Value) {
					return asc
				}

				return *propI.Value < *propJ.Value
			}
		})
	}
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

func generateTree(components models.Components, compChildrenMap map[string]models.Components) (models.Components, error) {
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
			if cc, err := generateTree(children, compChildrenMap); err != nil {
				return nil, err
			} else {
				c.Components = cc
			}
		}

		// TODO: Depth is added to prevent cyclic stackoverflow
		// Summary should be set after applyDepthFilter
		// which dereferences pointer cycles
		c.Summary = c.Summarize(10)

		if health, err := c.GetHealth(); err != nil {
			return nil, err
		} else {
			c.Health = lo.ToPtr(models.Health(health))
		}

		if status, err := c.GetStatus(); err != nil {
			return nil, err
		} else {
			c.Status = types.ComponentStatus(status)
		}

		nodes = append(nodes, c)
	}

	return nodes, nil
}

func createComponentTree(params TopologyOptions, components models.Components) ([]*models.Component, error) {
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

		for _, parentID := range c.Parents {
			compChildrenMap[parentID] = append(compChildrenMap[parentID], c)
		}

		// Keep a track of the root components for the current context
		// If params.ID is present only 1 root component can be there
		// else all components without a parent are root
		if params.ID != "" {
			if params.ID == c.ID.String() && len(rootComps) == 0 {
				rootComps = append(rootComps, c)
			}
		} else if c.ParentId == nil {
			rootComps = append(rootComps, c)
		}
	}

	return generateTree(rootComps, compChildrenMap)
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
func removeComponentFields(components models.Components) {
	for i := range components {
		c := components[i]

		c.ParentId = nil
		c.Parents = nil
		c.Checks = nil
		c.Incidents = nil
		c.Analysis = nil
		removeComponentFields(c.Components)
	}
}
