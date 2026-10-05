package tests

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

var _ = Describe("Scope membership", Ordered, func() {
	var (
		scopeID, wholeScopeID                uuid.UUID
		mtA, mtB, mtUpper, mtx, mtUnderscore models.ConfigItem
	)

	config := func(name string, tags map[string]string) models.ConfigItem {
		return models.ConfigItem{
			ID:          uuid.New(),
			Name:        lo.ToPtr(name),
			Type:        lo.ToPtr("Membership::Test"),
			ConfigClass: "Test",
			Tags:        tags,
		}
	}

	target := func(selector types.ResourceSelector) membership.Target {
		return membership.Target{Type: policy.ResourceConfig, Selector: selector}
	}

	scopesOf := func(item models.ConfigItem) []uuid.UUID {
		GinkgoHelper()
		snapshot, err := membership.Read(DefaultContext, membership.Ref{Type: policy.ResourceConfig, ID: item.ID})
		Expect(err).ToNot(HaveOccurred())
		return snapshot.Scopes(membership.Ref{Type: policy.ResourceConfig, ID: item.ID})
	}

	members := func(scope uuid.UUID) []uuid.UUID {
		GinkgoHelper()
		var ids []uuid.UUID
		Expect(DefaultContext.DB().Raw(`SELECT resource_id FROM scope_members WHERE scope_id = ? AND resource_id IS NOT NULL`, scope).
			Scan(&ids).Error).To(Succeed())
		return ids
	}

	// the targets stored on the test Scopes' rows; the targets passed to Rebuild stand in for their resolved form
	noTargets := types.JSON(`[]`)

	createScope := func(name string) uuid.UUID {
		GinkgoHelper()
		scope := models.Scope{ID: uuid.New(), Name: name, Namespace: "default", Targets: noTargets}
		Expect(DefaultContext.DB().Create(&scope).Error).To(Succeed())
		return scope.ID
	}

	BeforeAll(func() {
		mtA = config("mt-a", map[string]string{"team": "membership-test"})
		mtB = config("mt-b", map[string]string{"team": "other"})
		mtUpper = config("MT-c", map[string]string{"team": "membership-test"})
		mtx = config("mtx", map[string]string{"team": "membership-test"})
		mtUnderscore = config("mt_d", map[string]string{"team": "membership-test"})
		for _, item := range []models.ConfigItem{mtA, mtB, mtUpper, mtx, mtUnderscore} {
			Expect(DefaultContext.DB().Create(&item).Error).To(Succeed())
		}

		scopeID = createScope("membership-test")
		wholeScopeID = createScope("membership-test-all-configs")
	})

	AfterAll(func() {
		ids := []uuid.UUID{mtA.ID, mtB.ID, mtUpper.ID, mtx.ID, mtUnderscore.ID}
		Expect(DefaultContext.DB().Exec("DELETE FROM config_items WHERE id IN ?", ids).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM scopes WHERE id IN ?", []uuid.UUID{scopeID, wholeScopeID}).Error).To(Succeed())
	})

	It("matches names case-sensitively, with * only as a prefix and every other character literal", func() {
		rebuilt, err := membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{
			target(types.ResourceSelector{Name: "mt-*", TagSelector: "team=membership-test"}),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeTrue())
		Expect(members(scopeID)).To(Equal([]uuid.UUID{mtA.ID}))
	})

	It("rejects targets the Scope language doesn't have", func() {
		for _, selector := range []types.ResourceSelector{
			{Name: "*-a"},
			{Name: "m*-a"},
			{TagSelector: "team!=other"},
			{TagSelector: "team"},
			{TagSelector: "team in (a,b)"},
			{},
			{Name: "*", Statuses: types.Items{"healthy"}},
			{Name: "*", Health: "healthy"},
			{Name: "mt-a", Search: "env=prod"},
			{Name: "mt-a", FieldSelector: "type=x"},
			{Name: "mt-a", Scope: uuid.NewString()},
			{Name: "*", Functions: types.Functions{ComponentConfigTraversal: &types.ComponentConfigTraversalArgs{}}},
		} {
			Expect(membership.Validate(target(selector))).ToNot(Succeed(), "%+v", selector)
		}
		Expect(membership.Validate(membership.Target{Type: policy.ResourcePlaybook, Selector: types.ResourceSelector{Agent: uuid.NewString()}})).ToNot(Succeed())
		Expect(membership.Validate(target(types.ResourceSelector{Agent: "homelab"}))).ToNot(Succeed(), "agents must be resolved to ids")
		Expect(membership.Validate(target(types.ResourceSelector{TagSelector: "team==membership-test"}))).To(Succeed())
	})

	It("doesn't rebuild a Scope whose targets haven't changed", func() {
		rebuilt, err := membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{
			target(types.ResourceSelector{TagSelector: "team=membership-test", Name: "mt-*"}),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeFalse())
	})

	It("reads the Scopes a resource is in", func() {
		Expect(scopesOf(mtA)).To(ContainElement(scopeID))
		Expect(scopesOf(mtB)).ToNot(ContainElement(scopeID))
	})

	It("matches a changed resource in the transaction that changes it", func() {
		tx := DefaultContext.DB().Begin()
		Expect(tx.Model(&models.ConfigItem{}).Where("id = ?", mtB.ID).
			Update("tags", types.JSONStringMap{"team": "membership-test"}).Error).To(Succeed())
		Expect(scopesOf(mtB)).ToNot(ContainElement(scopeID), "other transactions see the previous membership until it commits")
		Expect(tx.Commit().Error).To(Succeed())

		Expect(scopesOf(mtB)).To(ContainElement(scopeID))
	})

	It("matches resources inserted in one statement, and re-matches them on upsert", func() {
		batch := []models.ConfigItem{
			config("mt-batch-1", map[string]string{"team": "membership-test"}),
			config("mt-batch-2", map[string]string{"team": "other"}),
		}
		Expect(DefaultContext.DB().Create(&batch).Error).To(Succeed())
		DeferCleanup(func() {
			Expect(DefaultContext.DB().Exec("DELETE FROM config_items WHERE id IN ?", []uuid.UUID{batch[0].ID, batch[1].ID}).Error).To(Succeed())
		})
		Expect(members(scopeID)).To(ContainElement(batch[0].ID))
		Expect(members(scopeID)).ToNot(ContainElement(batch[1].ID))

		Expect(DefaultContext.DB().Exec(`INSERT INTO config_items (id, name, type, config_class, tags) VALUES (?, 'mt-batch-2', 'Membership::Test', 'Test', '{"team":"membership-test"}')
			ON CONFLICT (id) DO UPDATE SET tags = excluded.tags`, batch[1].ID).Error).To(Succeed())
		Expect(members(scopeID)).To(ContainElement(batch[1].ID))
	})

	It("doesn't touch membership when a field a Scope can't select changes", func() {
		var before string
		Expect(DefaultContext.DB().Raw("SELECT xmin::text FROM scope_members WHERE scope_id = ? AND resource_id = ?", scopeID, mtA.ID).Scan(&before).Error).To(Succeed())

		Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", mtA.ID).Update("status", "Running").Error).To(Succeed())

		var after string
		Expect(DefaultContext.DB().Raw("SELECT xmin::text FROM scope_members WHERE scope_id = ? AND resource_id = ?", scopeID, mtA.ID).Scan(&after).Error).To(Succeed())
		Expect(after).To(Equal(before))
	})

	It("keeps a soft-deleted resource's membership", func() {
		Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", mtA.ID).
			Update("deleted_at", duty.Now()).Error).To(Succeed())
		Expect(scopesOf(mtA)).To(ContainElement(scopeID))
	})

	It("removes a hard-deleted resource's membership with the delete", func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM config_items WHERE id = ?", mtB.ID).Error).To(Succeed())
		Expect(members(scopeID)).ToNot(ContainElement(mtB.ID))
	})

	It("stores a whole-type target as one member with no resource, which admits every resource of the type", func() {
		_, err := membership.Rebuild(DefaultContext, wholeScopeID, noTargets, []membership.Target{
			target(types.ResourceSelector{Name: "*"}),
			target(types.ResourceSelector{Name: "mt-*"}),
			{Type: policy.ResourcePlaybook, Selector: types.ResourceSelector{Name: "mt-*"}},
		})
		Expect(err).ToNot(HaveOccurred())

		var stored int64
		Expect(DefaultContext.DB().Raw("SELECT COUNT(*) FROM scope_members WHERE scope_id = ? AND resource_type = 'config'", wholeScopeID).
			Scan(&stored).Error).To(Succeed())
		Expect(stored).To(Equal(int64(1)))

		Expect(scopesOf(mtx)).To(ContainElement(wholeScopeID))
		Expect(scopesOf(mtUpper)).To(ContainElement(wholeScopeID))

		var contained bool
		Expect(DefaultContext.DB().Raw("SELECT scope_contains(?, 'config', ?)", wholeScopeID, mtx.ID).Scan(&contained).Error).To(Succeed())
		Expect(contained).To(BeTrue())
		Expect(DefaultContext.DB().Raw("SELECT scope_contains(?, 'config', ?)", scopeID, mtx.ID).Scan(&contained).Error).To(Succeed())
		Expect(contained).To(BeFalse())
	})

	It("rebuilds a changed Scope writing only the members that change", func() {
		_, err := membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{
			target(types.ResourceSelector{Name: "mt-a"}),
			target(types.ResourceSelector{Name: "mt_*"}),
		})
		Expect(err).ToNot(HaveOccurred())

		var before string
		Expect(DefaultContext.DB().Raw("SELECT xmin::text FROM scope_members WHERE scope_id = ? AND resource_id = ?", scopeID, mtA.ID).Scan(&before).Error).To(Succeed())

		_, err = membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{
			target(types.ResourceSelector{Name: "mt-a"}),
			target(types.ResourceSelector{Name: "mtx"}),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(members(scopeID)).To(ConsistOf(mtA.ID, mtx.ID))

		var after string
		Expect(DefaultContext.DB().Raw("SELECT xmin::text FROM scope_members WHERE scope_id = ? AND resource_id = ?", scopeID, mtA.ID).Scan(&after).Error).To(Succeed())
		Expect(after).To(Equal(before), "a member the rebuild keeps isn't rewritten")
	})

	It("fails a rebuild that can't take the lock, leaving the previous membership", func() {
		timeout, retries, backoff := membership.LockTimeout, membership.LockRetries, membership.LockBackoff
		membership.LockTimeout, membership.LockRetries, membership.LockBackoff = 100*time.Millisecond, 1, 10*time.Millisecond
		DeferCleanup(func() {
			membership.LockTimeout, membership.LockRetries, membership.LockBackoff = timeout, retries, backoff
		})

		writer := DefaultContext.DB().Begin()
		Expect(writer.Exec("SELECT pg_advisory_xact_lock_shared(hashtext('scope_membership'))").Error).To(Succeed())

		_, err := membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{target(types.ResourceSelector{Name: "mt-b"})})
		Expect(err).To(MatchError(ContainSubstring(membership.ErrLockTimeout.Error())))
		Expect(writer.Rollback().Error).To(Succeed())
		Expect(members(scopeID)).To(ConsistOf(mtA.ID, mtx.ID))
	})

	It("refuses to rebuild from a version of the Scope that a newer save replaced", func() {
		newer := types.JSON(`[{"config": {"name": "mt-a"}}]`)
		Expect(DefaultContext.DB().Model(&models.Scope{}).Where("id = ?", scopeID).Update("targets", newer).Error).To(Succeed())
		DeferCleanup(func() {
			Expect(DefaultContext.DB().Model(&models.Scope{}).Where("id = ?", scopeID).Update("targets", noTargets).Error).To(Succeed())
		})

		rebuilt, err := membership.Rebuild(DefaultContext, scopeID, noTargets, []membership.Target{target(types.ResourceSelector{Name: "mt-b"})})
		Expect(err).To(MatchError(membership.ErrStaleScope))
		Expect(rebuilt).To(BeFalse())
		Expect(members(scopeID)).To(ConsistOf(mtA.ID, mtx.ID), "the newer version's membership is left alone")

		rebuilt, err = membership.Rebuild(DefaultContext, scopeID, newer, []membership.Target{target(types.ResourceSelector{Name: "mt-a"})})
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeTrue())
		Expect(members(scopeID)).To(ConsistOf(mtA.ID))
	})

	It("admits nothing through a deleted Scope", func() {
		Expect(membership.Clear(DefaultContext, scopeID)).To(Succeed())
		Expect(scopesOf(mtx)).ToNot(ContainElement(scopeID))

		has, err := membership.IsBuilt(DefaultContext, scopeID)
		Expect(err).ToNot(HaveOccurred())
		Expect(has).To(BeFalse())
	})

	It("deletes the membership of a Scope deleted outright", func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM scopes WHERE id = ?", wholeScopeID).Error).To(Succeed())
		Expect(scopesOf(mtx)).ToNot(ContainElement(wholeScopeID))
	})

	It("reads every resource of an operation in one snapshot", func() {
		ctx, err := membership.ForOperation(DefaultContext,
			membership.Ref{Type: policy.ResourceConfig, ID: mtA.ID},
			membership.Ref{Type: policy.ResourceConfig, ID: mtx.ID})
		Expect(err).ToNot(HaveOccurred())

		snapshot := membership.SnapshotFrom(ctx)
		Expect(snapshot.Covers(membership.Ref{Type: policy.ResourceConfig, ID: mtA.ID}, membership.Ref{Type: policy.ResourceConfig, ID: mtx.ID})).To(BeTrue())
		Expect(snapshot.Covers(membership.Ref{Type: policy.ResourceConfig, ID: mtB.ID})).To(BeFalse())
	})
})
