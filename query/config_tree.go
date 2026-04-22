package query

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

type ConfigTreeNode struct {
	models.ConfigItem `json:",inline"`
	EdgeType          string            `json:"edgeType,omitempty"`
	Relation          string            `json:"relation,omitempty"`
	Children          []*ConfigTreeNode `json:"children,omitempty"`
}

type ConfigTreeOptions struct {
	Direction RelationDirection
	Incoming  RelationType
	Outgoing  RelationType
}

func (n *ConfigTreeNode) OutgoingIDs() []uuid.UUID {
	var ids []uuid.UUID
	n.collectOutgoing(&ids, make(map[uuid.UUID]bool))
	return ids
}

func (n *ConfigTreeNode) collectOutgoing(ids *[]uuid.UUID, seen map[uuid.UUID]bool) {
	if seen[n.ID] {
		return
	}
	seen[n.ID] = true
	if n.EdgeType != "parent" {
		*ids = append(*ids, n.ID)
	}
	for _, c := range n.Children {
		c.collectOutgoing(ids, seen)
	}
}

func ConfigTree(ctx context.Context, configID uuid.UUID, opts ConfigTreeOptions) (*ConfigTreeNode, error) {
	config, err := GetCachedConfig(ctx, configID.String())
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}

	parents, err := resolveParentsFromPath(ctx, config)
	if err != nil {
		return nil, err
	}

	childIDs, err := ExpandConfigChildren(ctx, []uuid.UUID{config.ID})
	if err != nil {
		return nil, err
	}
	childIDs = lo.Filter(childIDs, func(id uuid.UUID, _ int) bool { return id != config.ID })

	var children []models.ConfigItem
	if len(childIDs) > 0 {
		children, err = GetConfigsByIDs(ctx, childIDs)
		if err != nil {
			return nil, err
		}
	}

	if opts.Direction == "" {
		opts.Direction = All
	}
	relType := Hard
	if opts.Incoming != "" {
		relType = opts.Incoming
	}
	outType := Both
	if opts.Outgoing != "" {
		outType = opts.Outgoing
	}

	related, err := GetRelatedConfigs(ctx, RelationQuery{
		ID:       config.ID,
		Relation: opts.Direction,
		Incoming: relType,
		Outgoing: outType,
	})
	if err != nil {
		return nil, err
	}

	return buildConfigTree(config, parents, children, related), nil
}

func resolveParentsFromPath(ctx context.Context, config *models.ConfigItem) ([]models.ConfigItem, error) {
	if config.Path == "" {
		return nil, nil
	}
	segments := strings.Split(config.Path, ".")
	var parentIDs []uuid.UUID
	for _, seg := range segments {
		id, err := uuid.Parse(seg)
		if err != nil || id == config.ID {
			continue
		}
		parentIDs = append(parentIDs, id)
	}
	if len(parentIDs) == 0 {
		return nil, nil
	}
	items, err := GetConfigsByIDs(ctx, parentIDs)
	if err != nil {
		return nil, fmt.Errorf("resolving parents from path: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	byID := make(map[uuid.UUID]models.ConfigItem, len(items))
	for _, ci := range items {
		byID[ci.ID] = ci
	}
	var parents []models.ConfigItem
	for _, id := range parentIDs {
		if ci, ok := byID[id]; ok {
			parents = append(parents, ci)
		}
	}
	return parents, nil
}

func ExpandConfigChildren(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	allIDs := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		allIDs[id] = struct{}{}
	}
	for _, id := range ids {
		var children []uuid.UUID
		if err := ctx.DB().Raw("SELECT child_id FROM lookup_config_children(?, -1)", id.String()).
			Scan(&children).Error; err != nil {
			return nil, err
		}
		for _, child := range children {
			allIDs[child] = struct{}{}
		}
	}
	return lo.Keys(allIDs), nil
}

type ptrNode struct {
	models.ConfigItem
	edgeType string
	relation string
	children []*ptrNode
}

func buildConfigTree(config *models.ConfigItem, parents []models.ConfigItem, children []models.ConfigItem, related []RelatedConfig) *ConfigTreeNode {
	nodes := make(map[uuid.UUID]*ptrNode)

	for _, p := range parents {
		nodes[p.ID] = &ptrNode{ConfigItem: p, edgeType: "parent"}
	}

	targetNode := &ptrNode{ConfigItem: *config, edgeType: "target"}
	nodes[config.ID] = targetNode

	if len(parents) > 0 {
		for i := 0; i < len(parents)-1; i++ {
			nodes[parents[i].ID].children = append(nodes[parents[i].ID].children, nodes[parents[i+1].ID])
		}
		nodes[parents[len(parents)-1].ID].children = append(nodes[parents[len(parents)-1].ID].children, targetNode)
	}

	for _, c := range children {
		nodes[c.ID] = &ptrNode{ConfigItem: c, edgeType: "child"}
	}

	parentIDs := make(map[uuid.UUID]bool, len(parents))
	for _, p := range parents {
		parentIDs[p.ID] = true
	}

	// wired tracks nodes that have already been attached to a parent so we never
	// append the same child twice (which would cause duplicate rendering).
	wired := make(map[uuid.UUID]bool)

	// Attach children by real ParentID first; if that parent isn't in our map,
	// walk the Path from nearest to furthest ancestor. Fall back to the target
	// only when no ancestor was selected.
	for _, c := range children {
		if wired[c.ID] {
			continue
		}
		if parent := resolveChildParent(c, nodes, config.ID); parent != nil {
			parent.children = append(parent.children, nodes[c.ID])
			wired[c.ID] = true
			continue
		}
		targetNode.children = append(targetNode.children, nodes[c.ID])
		wired[c.ID] = true
	}

	for _, rc := range related {
		if parentIDs[rc.ID] || rc.ID == config.ID {
			continue
		}
		if _, exists := nodes[rc.ID]; exists {
			// Already present as a parent/child/target — don't duplicate it as
			// a related node.
			continue
		}
		nodes[rc.ID] = &ptrNode{
			ConfigItem: relatedToConfigItem(rc),
			edgeType:   "related",
			relation:   rc.Relation,
		}
	}

	attachRelatedByAdjacency(targetNode, related, nodes, parentIDs, wired, config.ID)

	var root *ptrNode
	if len(parents) > 0 {
		root = nodes[parents[0].ID]
	} else {
		root = targetNode
	}

	return toConfigTreeNode(root, make(map[*ptrNode]bool))
}

// attachRelatedByAdjacency wires related nodes into a hierarchical tree using
// each RelatedConfig.RelatedIDs (the outgoing-edge set computed by
// related_configs_recursive). This places e.g. a SecurityGroup under the DB
// Instance that points at it, instead of flat under the target.
//
// Algorithm: build a reverse adjacency (child -> candidate parents), then BFS
// from the target, attaching each child to the first parent that itself got
// attached. Nodes never discovered via BFS are orphans and attach to target.
func attachRelatedByAdjacency(
	targetNode *ptrNode,
	related []RelatedConfig,
	nodes map[uuid.UUID]*ptrNode,
	parentIDs map[uuid.UUID]bool,
	wired map[uuid.UUID]bool,
	targetID uuid.UUID,
) {
	// outgoing[src] = dsts discovered via related_ids. src may or may not have
	// a ptrNode (the target itself has no row in `related`).
	outgoing := make(map[uuid.UUID][]uuid.UUID, len(related))
	for _, rc := range related {
		if len(rc.RelatedIDs) == 0 {
			continue
		}
		for _, s := range rc.RelatedIDs {
			child, err := uuid.Parse(s)
			if err != nil || child == rc.ID {
				continue
			}
			outgoing[rc.ID] = append(outgoing[rc.ID], child)
		}
	}

	// Reverse-map: for each child, which nodes point at it?
	incoming := make(map[uuid.UUID][]uuid.UUID)
	for src, dsts := range outgoing {
		for _, dst := range dsts {
			incoming[dst] = append(incoming[dst], src)
		}
	}

	// BFS from target, attaching nodes as we discover them. The target's own
	// RelatedIDs aren't in `related`, so we seed the queue with any related
	// node whose ID appears in some other node's RelatedIDs pointing out from
	// target — but since the target row is filtered out upstream, we fall back
	// to: any related node that has no incoming edge from another related node
	// is a direct child of target.
	queue := make([]uuid.UUID, 0, len(related))
	seeded := make(map[uuid.UUID]bool)
	for _, rc := range related {
		if parentIDs[rc.ID] || rc.ID == targetID || wired[rc.ID] {
			continue
		}
		if nodes[rc.ID] == nil || nodes[rc.ID].edgeType != "related" {
			continue
		}
		if len(incoming[rc.ID]) == 0 {
			queue = append(queue, rc.ID)
			seeded[rc.ID] = true
		}
	}

	attach := func(parent *ptrNode, childID uuid.UUID) bool {
		if wired[childID] || parentIDs[childID] || childID == targetID {
			return false
		}
		node := nodes[childID]
		if node == nil || node.edgeType != "related" {
			return false
		}
		parent.children = append(parent.children, node)
		wired[childID] = true
		return true
	}

	for _, id := range queue {
		attach(targetNode, id)
	}

	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]
		parentNode := nodes[parentID]
		if parentNode == nil {
			continue
		}
		for _, childID := range outgoing[parentID] {
			if attach(parentNode, childID) {
				queue = append(queue, childID)
			}
		}
	}

	// Any remaining related node that wasn't reached (cyclic incoming edges,
	// or pointed at only by nodes we couldn't attach) falls back to target.
	for _, rc := range related {
		if parentIDs[rc.ID] || rc.ID == targetID || wired[rc.ID] {
			continue
		}
		node := nodes[rc.ID]
		if node == nil || node.edgeType != "related" {
			continue
		}
		attach(targetNode, rc.ID)
	}
}

// resolveChildParent picks the best parent for a descendant config. Priority:
//  1. The node matching c.ParentID, if present.
//  2. The nearest ancestor along c.Path that is itself in the node map and is
//     not the target (so grandchildren nest under their real parent instead of
//     collapsing flat under the target).
//  3. The target node, if the path hits it.
//
// Returns nil if no suitable parent was found, in which case the caller
// attaches to the target as a last resort.
func resolveChildParent(c models.ConfigItem, nodes map[uuid.UUID]*ptrNode, targetID uuid.UUID) *ptrNode {
	if parentID := lo.FromPtr(c.ParentID); parentID != uuid.Nil {
		if parent, ok := nodes[parentID]; ok && parent.ID != c.ID {
			return parent
		}
	}
	if c.Path == "" {
		return nil
	}
	segments := strings.Split(c.Path, ".")
	var targetMatch *ptrNode
	for i := len(segments) - 1; i >= 0; i-- {
		pid, err := uuid.Parse(segments[i])
		if err != nil || pid == c.ID {
			continue
		}
		parent, ok := nodes[pid]
		if !ok {
			continue
		}
		if pid == targetID {
			targetMatch = parent
			continue
		}
		return parent
	}
	return targetMatch
}

func toConfigTreeNode(n *ptrNode, visited map[*ptrNode]bool) *ConfigTreeNode {
	result := &ConfigTreeNode{
		ConfigItem: n.ConfigItem,
		EdgeType:   n.edgeType,
		Relation:   n.relation,
	}
	if visited[n] {
		return result
	}
	visited[n] = true
	for _, c := range n.children {
		result.Children = append(result.Children, toConfigTreeNode(c, visited))
	}
	sort.SliceStable(result.Children, func(i, j int) bool {
		a, b := result.Children[i], result.Children[j]
		at, bt := lo.FromPtr(a.Type), lo.FromPtr(b.Type)
		if at != bt {
			return at < bt
		}
		return lo.FromPtr(a.Name) < lo.FromPtr(b.Name)
	})
	return result
}

func relatedToConfigItem(rc RelatedConfig) models.ConfigItem {
	ci := models.ConfigItem{
		ID: rc.ID,
	}
	ci.Name = &rc.Name
	ci.Type = &rc.Type
	ci.Tags = rc.Tags
	ci.Status = rc.Status
	ci.Health = rc.Health
	ci.Path = rc.Path
	ci.CreatedAt = rc.CreatedAt
	ci.UpdatedAt = &rc.UpdatedAt
	ci.DeletedAt = rc.DeletedAt
	ci.AgentID = rc.AgentID
	return ci
}
