package query

import (
	"fmt"
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
	for _, c := range children {
		parentID := lo.FromPtr(c.ParentID)
		if parent, ok := nodes[parentID]; ok {
			parent.children = append(parent.children, nodes[c.ID])
			continue
		}
		if c.Path != "" {
			if parent := findNearestAncestor(c.Path, nodes); parent != nil {
				parent.children = append(parent.children, nodes[c.ID])
				continue
			}
		}
		targetNode.children = append(targetNode.children, nodes[c.ID])
	}

	parentIDs := make(map[uuid.UUID]bool, len(parents))
	for _, p := range parents {
		parentIDs[p.ID] = true
	}

	for _, rc := range related {
		if parentIDs[rc.ID] || rc.ID == config.ID {
			continue
		}
		if _, exists := nodes[rc.ID]; !exists {
			nodes[rc.ID] = &ptrNode{
				ConfigItem: relatedToConfigItem(rc),
				edgeType:   "related",
				relation:   rc.Relation,
			}
		}
	}

	wired := make(map[uuid.UUID]bool)
	for _, rc := range related {
		if parentIDs[rc.ID] || rc.ID == config.ID || wired[rc.ID] {
			continue
		}
		wired[rc.ID] = true
		node := nodes[rc.ID]
		if rc.Path != "" {
			if parent := findNearestAncestor(rc.Path, nodes); parent != nil && parent != node && !parentIDs[parent.ID] {
				parent.children = append(parent.children, node)
				continue
			}
		}
		targetNode.children = append(targetNode.children, node)
	}

	var root *ptrNode
	if len(parents) > 0 {
		root = nodes[parents[0].ID]
	} else {
		root = targetNode
	}

	return toConfigTreeNode(root, make(map[*ptrNode]bool))
}

func findNearestAncestor(path string, nodes map[uuid.UUID]*ptrNode) *ptrNode {
	segments := strings.Split(path, ".")
	for i := len(segments) - 1; i >= 0; i-- {
		if pid, err := uuid.Parse(segments[i]); err == nil {
			if parent, ok := nodes[pid]; ok {
				return parent
			}
		}
	}
	return nil
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
