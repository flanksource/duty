package tests

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/types"
)

var _ = Describe("Scope membership exclusions", Ordered, func() {
	var (
		scope                         models.Scope
		run, deleteProd, deleteDev    models.Playbook
		breakGlass, renamed, newcomer models.Playbook
	)

	playbook := func(name, namespace string) models.Playbook {
		return models.Playbook{ID: uuid.New(), Name: name, Namespace: namespace, Spec: types.JSON(`{}`), Source: models.SourceUI}
	}

	target := func(selector types.ResourceSelector) membership.Target {
		return membership.Target{Type: policy.ResourcePlaybook, Selector: selector}
	}
	exclusion := func(selector types.ResourceSelector) membership.Target {
		return membership.Target{Type: policy.ResourcePlaybook, Selector: selector, Exclude: true}
	}

	// everyPlaybookExcept is the customer's Scope: every playbook except the delete-* ones and those in mse-break-glass.
	everyPlaybookExcept := []membership.Target{
		target(types.ResourceSelector{Name: "*"}),
		exclusion(types.ResourceSelector{Name: "mse-delete-*"}),
		exclusion(types.ResourceSelector{Namespace: "mse-break-glass"}),
	}

	in := func(p models.Playbook) bool {
		GinkgoHelper()
		snapshot, err := membership.Read(DefaultContext, membership.Ref{Type: policy.ResourcePlaybook, ID: p.ID})
		Expect(err).ToNot(HaveOccurred())
		for _, id := range snapshot.Scopes(membership.Ref{Type: policy.ResourcePlaybook, ID: p.ID}) {
			if id == scope.ID {
				return true
			}
		}
		return false
	}

	matches := func(p models.Playbook) bool {
		GinkgoHelper()
		var matched bool
		Expect(DefaultContext.DB().Raw("SELECT _scope_matches(?, 'playbook', ?, ?, ?, NULL, NULL, NULL, NULL)",
			scope.ID, p.ID, p.Name, p.Namespace).Scan(&matched).Error).To(Succeed())
		return matched
	}

	wholeTypeRows := func() int64 {
		GinkgoHelper()
		var count int64
		Expect(DefaultContext.DB().Raw("SELECT COUNT(*) FROM scope_members WHERE scope_id = ? AND resource_id IS NULL", scope.ID).
			Scan(&count).Error).To(Succeed())
		return count
	}

	BeforeAll(func() {
		run = playbook("mse-run", "mse-default")
		deleteProd = playbook("mse-delete-production", "mse-default")
		deleteDev = playbook("mse-delete-dev", "mse-staging")
		breakGlass = playbook("mse-rotate-credentials", "mse-break-glass")
		renamed = playbook("mse-restart", "mse-default")
		for _, p := range []models.Playbook{run, deleteProd, deleteDev, breakGlass, renamed} {
			Expect(DefaultContext.DB().Create(&p).Error).To(Succeed())
		}

		scope = models.Scope{ID: uuid.New(), Name: "mse-every-playbook-except", Namespace: "default", Targets: types.JSON(`[]`), Exclude: types.JSON(`[]`)}
		Expect(DefaultContext.DB().Create(&scope).Error).To(Succeed())
	})

	AfterAll(func() {
		ids := []uuid.UUID{run.ID, deleteProd.ID, deleteDev.ID, breakGlass.ID, renamed.ID}
		if newcomer.ID != uuid.Nil {
			ids = append(ids, newcomer.ID)
		}
		Expect(DefaultContext.DB().Exec("DELETE FROM playbooks WHERE id IN ?", ids).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM scopes WHERE id = ?", scope.ID).Error).To(Succeed())
	})

	It("doesn't treat a whole-type target as the whole type when the Scope excludes resources of it", func() {
		Expect(membership.WholeTypes(everyPlaybookExcept)).To(BeEmpty())
		Expect(membership.WholeTypes(everyPlaybookExcept[:1])).To(Equal([]string{policy.ResourcePlaybook}))

		rebuilt, err := membership.RebuildScope(DefaultContext, scope, everyPlaybookExcept)
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeTrue())
		Expect(wholeTypeRows()).To(BeZero())
	})

	It("admits every resource a target matches except those an exclusion matches", func() {
		Expect(in(run)).To(BeTrue())
		Expect(in(renamed)).To(BeTrue())
		Expect(in(deleteProd)).To(BeFalse(), "excluded by name prefix")
		Expect(in(deleteDev)).To(BeFalse(), "excluded by name prefix in any namespace")
		Expect(in(breakGlass)).To(BeFalse(), "excluded by namespace")

		Expect(matches(run)).To(BeTrue())
		Expect(matches(deleteProd)).To(BeFalse())
		Expect(matches(breakGlass)).To(BeFalse())
	})

	It("doesn't admit a new resource that matches an exclusion", func() {
		newcomer = playbook("mse-delete-everything", "mse-default")
		Expect(DefaultContext.DB().Create(&newcomer).Error).To(Succeed())
		Expect(in(newcomer)).To(BeFalse())

		other := playbook("mse-scale", "mse-default")
		Expect(DefaultContext.DB().Create(&other).Error).To(Succeed())
		DeferCleanup(func() { Expect(DefaultContext.DB().Delete(&other).Error).To(Succeed()) })
		Expect(in(other)).To(BeTrue())
	})

	It("removes a member renamed into an exclusion, and admits it renamed out", func() {
		Expect(DefaultContext.DB().Model(&models.Playbook{}).Where("id = ?", renamed.ID).Update("name", "mse-delete-restart").Error).To(Succeed())
		Expect(in(renamed)).To(BeFalse())

		Expect(DefaultContext.DB().Model(&models.Playbook{}).Where("id = ?", renamed.ID).Update("name", "mse-restart").Error).To(Succeed())
		Expect(in(renamed)).To(BeTrue())
	})

	It("doesn't rebuild a Scope whose targets and exclusions haven't changed", func() {
		rebuilt, err := membership.RebuildScope(DefaultContext, scope, everyPlaybookExcept)
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeFalse())
	})

	It("refuses to rebuild from a version whose exclusions a newer save replaced", func() {
		newer := types.JSON(`[{"playbook": {"name": "mse-delete-*"}}]`)
		Expect(DefaultContext.DB().Model(&models.Scope{}).Where("id = ?", scope.ID).Update("exclude", newer).Error).To(Succeed())
		DeferCleanup(func() {
			Expect(DefaultContext.DB().Model(&models.Scope{}).Where("id = ?", scope.ID).Update("exclude", scope.Exclude).Error).To(Succeed())
		})

		_, err := membership.RebuildScope(DefaultContext, scope, everyPlaybookExcept[:2])
		Expect(err).To(MatchError(membership.ErrStaleScope))
		Expect(in(breakGlass)).To(BeFalse(), "the newer version's membership is left alone")
	})

	It("treats no exclusions alike however they're stored", func() {
		for _, stored := range []string{"NULL", "'null'::jsonb", "'[]'::jsonb"} {
			Expect(DefaultContext.DB().Exec("UPDATE scopes SET exclude = "+stored+" WHERE id = ?", scope.ID).Error).To(Succeed())
			for _, exclude := range []types.JSON{nil, types.JSON(`null`), types.JSON(`[]`)} {
				_, err := membership.RebuildScope(DefaultContext, models.Scope{ID: scope.ID, Targets: scope.Targets, Exclude: exclude}, everyPlaybookExcept)
				Expect(err).ToNot(HaveOccurred(), "stored %s, rebuilt with %q", stored, string(exclude))
			}
		}
	})

	It("stores the whole type again once its exclusions are removed", func() {
		rebuilt, err := membership.RebuildScope(DefaultContext, scope, everyPlaybookExcept[:1])
		Expect(err).ToNot(HaveOccurred())
		Expect(rebuilt).To(BeTrue())
		Expect(wholeTypeRows()).To(Equal(int64(1)))
		Expect(in(deleteProd)).To(BeTrue())
		Expect(in(breakGlass)).To(BeTrue())

		var members int64
		Expect(DefaultContext.DB().Raw("SELECT COUNT(*) FROM scope_members WHERE scope_id = ? AND resource_id IS NOT NULL", scope.ID).
			Scan(&members).Error).To(Succeed())
		Expect(members).To(BeZero())
	})

	It("rejects an exclusion of a whole type", func() {
		Expect(membership.Validate(exclusion(types.ResourceSelector{Name: "*"}))).ToNot(Succeed())
		Expect(membership.Validate(exclusion(types.ResourceSelector{Name: "mse-delete-*"}))).To(Succeed())

		_, err := membership.RebuildScope(DefaultContext, scope, []membership.Target{
			target(types.ResourceSelector{Name: "*"}),
			exclusion(types.ResourceSelector{Name: "*"}),
		})
		Expect(err).To(HaveOccurred())
	})
})
