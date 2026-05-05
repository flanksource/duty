package query

import (
	"strings"
	"testing"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/onsi/gomega"
)

func ptr[T any](v T) *T { return &v }

func makeCI(id, parentID uuid.UUID, name, typ string, ancestors ...uuid.UUID) models.ConfigItem {
	ci := models.ConfigItem{
		ID:   id,
		Name: &name,
		Type: &typ,
	}
	if parentID != uuid.Nil {
		ci.ParentID = &parentID
	}
	segments := make([]string, 0, len(ancestors)+1)
	for _, a := range ancestors {
		segments = append(segments, a.String())
	}
	segments = append(segments, id.String())
	ci.Path = strings.Join(segments, ".")
	return ci
}

func collectIDs(node *ConfigTreeNode) []uuid.UUID {
	if node == nil {
		return nil
	}
	out := []uuid.UUID{node.ID}
	for _, c := range node.Children {
		out = append(out, collectIDs(c)...)
	}
	return out
}

// TestBuildConfigTreeNestsGrandchildren reproduces the single-depth bug: a
// grandchild must appear under its real parent, not flattened under the target.
func TestBuildConfigTreeNestsGrandchildren(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	child := uuid.New()
	grand := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "region-a", "Geo::Country")
	childCI := makeCI(child, target, "city-a", "Geo::City", target)
	grandCI := makeCI(grand, child, "block-10", "Geo::District", target, child)

	tree := buildConfigTree(&targetCI, nil, []models.ConfigItem{childCI, grandCI}, nil)

	g.Expect(tree).ToNot(gomega.BeNil())
	g.Expect(tree.ID).To(gomega.Equal(target))
	g.Expect(tree.Children).To(gomega.HaveLen(1), "target should have exactly one direct child")
	g.Expect(tree.Children[0].ID).To(gomega.Equal(child))
	g.Expect(tree.Children[0].Children).To(gomega.HaveLen(1), "child should host the grandchild, not the target")
	g.Expect(tree.Children[0].Children[0].ID).To(gomega.Equal(grand))
}

// TestBuildConfigTreeDedupsChildren ensures a child with both a ParentID and a
// path hit against the target isn't attached twice.
func TestBuildConfigTreeDedupsChildren(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	child := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "region-a", "Geo::Country")
	childCI := makeCI(child, target, "city-a", "Geo::City", target)

	tree := buildConfigTree(&targetCI, nil, []models.ConfigItem{childCI}, nil)

	ids := collectIDs(tree)
	childCount := 0
	for _, id := range ids {
		if id == child {
			childCount++
		}
	}
	g.Expect(childCount).To(gomega.Equal(1), "child must not be attached twice")
}

// TestBuildConfigTreeSoftDoesNotDuplicateChildren reproduces the --soft bug:
// related rows that repeat hard child relationships must not render again
// under the target as "~ related" nodes.
func TestBuildConfigTreeSoftDoesNotDuplicateChildren(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	child := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "region-a", "Geo::Country")
	childCI := makeCI(child, target, "city-a", "Geo::City", target)

	// The related slice includes the same child that already appears in the
	// hard children slice — this is what `--soft` (RelationType=Both) does.
	related := []RelatedConfig{{
		ID:       child,
		Name:     "city-a",
		Type:     "Geo::City",
		Path:     childCI.Path,
		Relation: "contains",
	}}

	tree := buildConfigTree(&targetCI, nil, []models.ConfigItem{childCI}, related)

	g.Expect(tree.Children).To(gomega.HaveLen(1), "child must appear exactly once, not as both child and related")
	g.Expect(tree.Children[0].ID).To(gomega.Equal(child))
	g.Expect(tree.Children[0].EdgeType).To(gomega.Equal("child"), "child takes priority over related edge type")
}

// TestBuildConfigTreeRelatedStillAttachesStandaloneNodes verifies that related
// configs that are NOT already children still appear in the tree.
func TestBuildConfigTreeRelatedStillAttachesStandaloneNodes(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	rel := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "region-a", "Geo::Country")

	related := []RelatedConfig{{
		ID:       rel,
		Name:     "trade-partner",
		Type:     "Geo::Country",
		Relation: "trades-with",
	}}

	tree := buildConfigTree(&targetCI, nil, nil, related)

	g.Expect(tree.Children).To(gomega.HaveLen(1))
	g.Expect(tree.Children[0].ID).To(gomega.Equal(rel))
	g.Expect(tree.Children[0].EdgeType).To(gomega.Equal("related"))
	g.Expect(tree.Children[0].Relation).To(gomega.Equal("trades-with"))
}

// TestBuildConfigTreeNestsRelatedViaAdjacency reproduces the multi-hop case:
// a security group related to a DB Instance must nest under the DB Instance,
// not flat under the target country.
func TestBuildConfigTreeNestsRelatedViaAdjacency(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	env := uuid.New()
	db := uuid.New()
	sg := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "region-b", "Geo::Country")

	related := []RelatedConfig{
		// env is a direct child of target; it points at the DB.
		{
			ID:         env,
			Name:       "uat-region",
			Type:       "Geo::Environment",
			Relation:   "hosts",
			RelatedIDs: []string{db.String()},
		},
		// db points at the security group.
		{
			ID:         db,
			Name:       "shared-db",
			Type:       "AWS::RDS::DBInstance",
			Relation:   "runs",
			RelatedIDs: []string{sg.String()},
		},
		// security group — endpoint only, no further adjacency.
		{
			ID:       sg,
			Name:     "RDSAccess",
			Type:     "AWS::EC2::SecurityGroup",
			Relation: "RDSSecurityGroup",
		},
	}

	tree := buildConfigTree(&targetCI, nil, nil, related)

	g.Expect(tree.Children).To(gomega.HaveLen(1), "env should be the only direct child of target")
	envNode := tree.Children[0]
	g.Expect(envNode.ID).To(gomega.Equal(env))
	g.Expect(envNode.Children).To(gomega.HaveLen(1), "db should nest under env")
	dbNode := envNode.Children[0]
	g.Expect(dbNode.ID).To(gomega.Equal(db))
	g.Expect(dbNode.Children).To(gomega.HaveLen(1), "sg should nest under db")
	g.Expect(dbNode.Children[0].ID).To(gomega.Equal(sg))
}

// TestBuildConfigTreeOrphansFallBackToTarget ensures a related node pointed at
// only by nodes we can't reach ends up under target rather than being lost.
func TestBuildConfigTreeOrphansFallBackToTarget(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	a := uuid.New()
	b := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "t", "T")

	// a -> b forms a cycle with only incoming edges — neither is reachable
	// from target via RelatedIDs, so both must fall through to target.
	related := []RelatedConfig{
		{ID: a, Name: "a", Type: "T", RelatedIDs: []string{b.String()}},
		{ID: b, Name: "b", Type: "T", RelatedIDs: []string{a.String()}},
	}

	tree := buildConfigTree(&targetCI, nil, nil, related)

	ids := collectIDs(tree)
	g.Expect(ids).To(gomega.ContainElement(a))
	g.Expect(ids).To(gomega.ContainElement(b))
}

// TestBuildConfigTreeSortsChildrenByTypeThenName confirms every level is
// ordered by (type, name) regardless of input order.
func TestBuildConfigTreeSortsChildrenByTypeThenName(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	d := uuid.New()

	targetCI := makeCI(target, uuid.Nil, "t", "T")

	// Insertion order intentionally scrambled.
	related := []RelatedConfig{
		{ID: c, Name: "zebra", Type: "B"},
		{ID: a, Name: "alpha", Type: "A"},
		{ID: d, Name: "alpha", Type: "B"},
		{ID: b, Name: "beta", Type: "A"},
	}

	tree := buildConfigTree(&targetCI, nil, nil, related)

	g.Expect(tree.Children).To(gomega.HaveLen(4))
	g.Expect(tree.Children[0].ID).To(gomega.Equal(a), "A/alpha first")
	g.Expect(tree.Children[1].ID).To(gomega.Equal(b), "A/beta second")
	g.Expect(tree.Children[2].ID).To(gomega.Equal(d), "B/alpha third")
	g.Expect(tree.Children[3].ID).To(gomega.Equal(c), "B/zebra last")
}

// TestResolveChildParentPrefersParentID confirms ParentID lookup wins over
// path-walking.
func TestResolveChildParentPrefersParentID(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	parent := uuid.New()
	child := uuid.New()

	childCI := makeCI(child, parent, "x", "t", target, parent)

	nodes := map[uuid.UUID]*ptrNode{
		target: {ConfigItem: makeCI(target, uuid.Nil, "t", "t"), edgeType: "target"},
		parent: {ConfigItem: makeCI(parent, target, "p", "t", target), edgeType: "child"},
	}

	got := resolveChildParent(childCI, nodes, target)
	g.Expect(got).ToNot(gomega.BeNil())
	g.Expect(got.ID).To(gomega.Equal(parent))
}

// TestResolveChildParentFallsBackToPathWhenParentMissing verifies path-walk
// fallback finds the nearest ancestor that is not the target.
func TestResolveChildParentFallsBackToPathWhenParentMissing(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	missingParent := uuid.New()
	ancestor := uuid.New()
	child := uuid.New()

	// child's ParentID points at a node we never loaded; Path contains target
	// and ancestor.
	childCI := models.ConfigItem{
		ID:       child,
		ParentID: &missingParent,
		Name:     ptr("c"),
		Type:     ptr("t"),
		Path:     target.String() + "." + ancestor.String() + "." + child.String(),
	}

	nodes := map[uuid.UUID]*ptrNode{
		target:   {ConfigItem: makeCI(target, uuid.Nil, "t", "t"), edgeType: "target"},
		ancestor: {ConfigItem: makeCI(ancestor, target, "a", "t", target), edgeType: "child"},
	}

	got := resolveChildParent(childCI, nodes, target)
	g.Expect(got).ToNot(gomega.BeNil())
	g.Expect(got.ID).To(gomega.Equal(ancestor), "nearest non-target ancestor in path wins")
}

// TestResolveChildParentFallsThroughToTarget confirms target is the last
// resort when no non-target ancestor is in the map.
func TestResolveChildParentFallsThroughToTarget(t *testing.T) {
	g := gomega.NewWithT(t)

	target := uuid.New()
	child := uuid.New()

	childCI := makeCI(child, uuid.Nil, "c", "t", target)

	nodes := map[uuid.UUID]*ptrNode{
		target: {ConfigItem: makeCI(target, uuid.Nil, "t", "t"), edgeType: "target"},
	}

	got := resolveChildParent(childCI, nodes, target)
	g.Expect(got).ToNot(gomega.BeNil())
	g.Expect(got.ID).To(gomega.Equal(target))
}

func makeTreeNode(id uuid.UUID, name string, edge string, children ...*ConfigTreeNode) *ConfigTreeNode {
	n := name
	return &ConfigTreeNode{
		ConfigItem: models.ConfigItem{ID: id, Name: &n},
		EdgeType:   edge,
		Children:   children,
	}
}

// TestMergeConfigTreesSharedAncestor collapses two RDS matches into a single
// tree when they share an account and stack — the scenario that motivated the
// merge (two RDS instances under the same CloudFormation stack in one AWS
// account).
func TestMergeConfigTreesSharedAncestor(t *testing.T) {
	g := gomega.NewWithT(t)

	account := uuid.New()
	stack := uuid.New()
	rds1 := uuid.New()
	rds2 := uuid.New()

	tree1 := makeTreeNode(account, "acct", "parent",
		makeTreeNode(stack, "stack", "parent",
			makeTreeNode(rds1, "db1", "target"),
		),
	)
	tree2 := makeTreeNode(account, "acct", "parent",
		makeTreeNode(stack, "stack", "parent",
			makeTreeNode(rds2, "db2", "target"),
		),
	)

	roots := MergeConfigTrees([]*ConfigTreeNode{tree1, tree2})
	g.Expect(roots).To(gomega.HaveLen(1))
	g.Expect(roots[0].ID).To(gomega.Equal(account))
	g.Expect(roots[0].Children).To(gomega.HaveLen(1))
	g.Expect(roots[0].Children[0].ID).To(gomega.Equal(stack))
	g.Expect(roots[0].Children[0].Children).To(gomega.HaveLen(2))

	childIDs := []uuid.UUID{
		roots[0].Children[0].Children[0].ID,
		roots[0].Children[0].Children[1].ID,
	}
	g.Expect(childIDs).To(gomega.ConsistOf(rds1, rds2))
}

// TestMergeConfigTreesUnrelatedRootsStaySeparate confirms that two unrelated
// roots (e.g. two different AWS accounts) remain as sibling root trees.
func TestMergeConfigTreesUnrelatedRootsStaySeparate(t *testing.T) {
	g := gomega.NewWithT(t)

	acctA := uuid.New()
	acctB := uuid.New()
	stackA := uuid.New()
	stackB := uuid.New()
	dbA := uuid.New()
	dbB := uuid.New()

	trees := []*ConfigTreeNode{
		makeTreeNode(acctA, "A", "parent", makeTreeNode(stackA, "sA", "parent", makeTreeNode(dbA, "dbA", "target"))),
		makeTreeNode(acctB, "B", "parent", makeTreeNode(stackB, "sB", "parent", makeTreeNode(dbB, "dbB", "target"))),
	}

	roots := MergeConfigTrees(trees)
	g.Expect(roots).To(gomega.HaveLen(2))
	rootIDs := []uuid.UUID{roots[0].ID, roots[1].ID}
	g.Expect(rootIDs).To(gomega.ConsistOf(acctA, acctB))
}

// TestMergeConfigTreesPreservesTargetEdge ensures EdgeType="target" wins when
// the same node appears as both an ancestor (parent edge) in one tree and a
// target in another.
func TestMergeConfigTreesPreservesTargetEdge(t *testing.T) {
	g := gomega.NewWithT(t)

	root := uuid.New()
	target := uuid.New()

	t1 := makeTreeNode(root, "root", "parent", makeTreeNode(target, "t", "parent"))
	t2 := makeTreeNode(root, "root", "parent", makeTreeNode(target, "t", "target"))

	roots := MergeConfigTrees([]*ConfigTreeNode{t1, t2})
	g.Expect(roots).To(gomega.HaveLen(1))
	g.Expect(roots[0].Children).To(gomega.HaveLen(1))
	g.Expect(roots[0].Children[0].EdgeType).To(gomega.Equal("target"))
}

func TestMergeConfigTreesEmpty(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(MergeConfigTrees(nil)).To(gomega.BeEmpty())
	g.Expect(MergeConfigTrees([]*ConfigTreeNode{nil, nil})).To(gomega.BeEmpty())
}

// TestMergeConfigTreesSharedInternalNode confirms that a node shared between
// two trees with different roots has its children unioned, not lost. With the
// previous early-return in cloneConfigTree, the second tree's children under a
// shared internal node were silently dropped.
func TestMergeConfigTreesSharedInternalNode(t *testing.T) {
	g := gomega.NewWithT(t)

	a := uuid.New()
	e := uuid.New()
	b := uuid.New()
	d := uuid.New()
	x := uuid.New()

	tree1 := makeTreeNode(a, "a", "parent",
		makeTreeNode(b, "b", "parent",
			makeTreeNode(d, "d", "target"),
		),
	)
	tree2 := makeTreeNode(e, "e", "parent",
		makeTreeNode(b, "b", "parent",
			makeTreeNode(x, "x", "target"),
		),
	)

	roots := MergeConfigTrees([]*ConfigTreeNode{tree1, tree2})
	g.Expect(roots).To(gomega.HaveLen(2), "a and e have different roots, so two trees")

	rootByID := map[uuid.UUID]*ConfigTreeNode{}
	for _, r := range roots {
		rootByID[r.ID] = r
	}
	g.Expect(rootByID).To(gomega.HaveKey(a))
	g.Expect(rootByID).To(gomega.HaveKey(e))

	bUnderA := rootByID[a].Children[0]
	bUnderE := rootByID[e].Children[0]
	g.Expect(bUnderA.ID).To(gomega.Equal(b))
	g.Expect(bUnderE.ID).To(gomega.Equal(b))
	g.Expect(bUnderA).To(gomega.BeIdenticalTo(bUnderE), "b should be the same node aliased under both roots")

	childIDs := []uuid.UUID{}
	for _, c := range bUnderA.Children {
		childIDs = append(childIDs, c.ID)
	}
	g.Expect(childIDs).To(gomega.ConsistOf(d, x), "both d and x must appear under merged b")
}
