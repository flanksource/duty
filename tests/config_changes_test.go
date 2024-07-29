package tests

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Config changes recursive", ginkgo.Ordered, func() {
	// Graph #1 (acyclic)
	//
	//        U
	//       / \
	//      V   W
	//     / \
	//    X   Y
	//   /
	//  Z

	// Create a list of ConfigItems
	var (
		U = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Node"), Name: lo.ToPtr("U"), ConfigClass: "Node"}
		V = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Deployment"), Name: lo.ToPtr("V"), ConfigClass: "Deployment"}
		W = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Pod"), Name: lo.ToPtr("W"), ConfigClass: "Pod"}
		X = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ReplicaSet"), Name: lo.ToPtr("X"), ConfigClass: "ReplicaSet"}
		Y = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::PersistentVolume"), Name: lo.ToPtr("Y"), ConfigClass: "PersistentVolume"}
		Z = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Pod"), Name: lo.ToPtr("Z"), ConfigClass: "Pod"}
	)
	configItems := []models.ConfigItem{U, V, W, X, Y, Z}

	// Create relationships between ConfigItems
	relationships := []models.ConfigRelationship{
		{ConfigID: U.ID.String(), RelatedID: V.ID.String(), Relation: "test-changes-UV"},
		{ConfigID: U.ID.String(), RelatedID: W.ID.String(), Relation: "test-changes-UW"},
		{ConfigID: V.ID.String(), RelatedID: X.ID.String(), Relation: "test-changes-VX"},
		{ConfigID: V.ID.String(), RelatedID: Y.ID.String(), Relation: "test-changes-VY"},
		{ConfigID: X.ID.String(), RelatedID: Z.ID.String(), Relation: "test-changes-XZ"},
	}

	// Create changes for each config
	var (
		UChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now()), Severity: "info", ConfigID: U.ID.String(), Summary: ".name.U", ChangeType: "RegisterNode", Source: "test-changes"}
		VChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour)), Severity: models.SeverityHigh, ConfigID: V.ID.String(), Summary: ".name.V", ChangeType: "diff", Source: "test-changes"}
		WChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 2)), Severity: models.SeverityCritical, ConfigID: W.ID.String(), Summary: ".name.W", ChangeType: "Pulled", Source: "test-changes"}
		XChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 3)), Severity: "info", ConfigID: X.ID.String(), Summary: ".name.X", ChangeType: "diff", Source: "test-changes"}
		YChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 4)), Severity: "warn", ConfigID: Y.ID.String(), Summary: ".name.Y", ChangeType: "diff", Source: "test-changes"}
		ZChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: Z.ID.String(), Summary: ".name.Z", ChangeType: "Pulled", Source: "test-changes"}

		changes = []models.ConfigChange{UChange, VChange, WChange, XChange, YChange, ZChange}
	)

	ginkgo.BeforeAll(func() {

		DefaultContext = DefaultContext.WithDBLogLevel("debug").WithTrace()

		// Save configs
		err := DefaultContext.DB().Create(&configItems).Error
		Expect(err).To(BeNil())

		var foundConfigs []models.ConfigItem
		err = DefaultContext.DB().Select("id").Where("tags->>'namespace' = 'test-changes'").Find(&foundConfigs).Error
		Expect(err).To(BeNil())
		Expect(len(foundConfigs)).To(Equal(len(configItems)))

		// Save relationships
		err = DefaultContext.DB().Create(&relationships).Error
		Expect(err).To(BeNil())

		var foundRelationships []models.ConfigRelationship
		err = DefaultContext.DB().Where("relation LIKE 'test-changes%'").Find(&foundRelationships).Error
		Expect(err).To(BeNil())
		Expect(len(foundRelationships)).To(Equal(len(relationships)))

		// Save changes
		err = DefaultContext.DB().Create(&changes).Error
		Expect(err).To(BeNil())

		var foundChanges []models.ConfigChange
		err = DefaultContext.DB().Where("source = 'test-changes'").Find(&foundChanges).Error
		Expect(err).To(BeNil())
		Expect(len(foundChanges)).To(Equal(len(changes)))
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Where("relation LIKE 'test-changes%'").Delete(&models.ConfigRelationship{}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Where("source = 'test-changes'").Delete(&models.ConfigChange{}).Error
		Expect(err).To(BeNil())

		err = DefaultContext.DB().Where("tags->>'namespace' = 'test-changes'").Delete(&models.ConfigItem{}).Error
		Expect(err).To(BeNil())
	})

	var findChanges = func(id uuid.UUID, filter string, deleted bool) (*query.CatalogChangesSearchResponse, error) {
		return query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
			CatalogID:             id.String(),
			IncludeDeletedConfigs: deleted,
			Recursive:             filter,
			Depth:                 5,
		})
	}

	ginkgo.Context("Both ways", func() {
		ginkgo.It("should return changes upstream and downstream", func() {
			relatedChanges, err := findChanges(X.ID, "all", false)

			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(4))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, XChange.ID, ZChange.ID}))
		})
	})

	ginkgo.Context("Downstream", func() {
		ginkgo.It("should return changes of a root node", func() {
			relatedChanges, err := findChanges(U.ID, "downstream", false)
			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(6))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, WChange.ID, XChange.ID, YChange.ID, ZChange.ID}))
		})

		ginkgo.It("should return changes of a leaf node", func() {
			relatedChanges, err := findChanges(Z.ID, "all", false)
			Expect(err).To(BeNil())
			Expect(len(relatedChanges.Changes)).To(Equal(4))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{ZChange.ID, XChange.ID, VChange.ID, UChange.ID}))
		})
	})

	ginkgo.Context("Upstream", func() {
		ginkgo.It("should return changes for a root node", func() {
			relatedChanges, err := findChanges(U.ID, "upstream", false)
			Expect(err).To(BeNil())
			Expect(len(relatedChanges.Changes)).To(Equal(1))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID}))
		})

		ginkgo.It("should return changes of a non-root node", func() {
			relatedChanges, err := findChanges(X.ID, "all", false)
			Expect(err).To(BeNil())
			Expect(len(relatedChanges.Changes)).To(Equal(4))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, ZChange.ID, XChange.ID}))
		})
	})

	ginkgo.Context("FindCatalogChanges func", func() {
		ginkgo.It("Without catalog id", func() {
			response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				ConfigType: "Kubernetes::Pod,Kubernetes::ReplicaSet",
			})
			Expect(err).To(BeNil())

			Expect(response.Total).To(Equal(int64(3)))
			Expect(len(response.Changes)).To(Equal(3))
			Expect(response.Summary["Pulled"]).To(Equal(2))
			Expect(response.Summary["diff"]).To(Equal(1))
		})

		ginkgo.Context("Config type filter", func() {
			ginkgo.It("IN", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID:  U.ID.String(),
					Recursive:  query.CatalogChangeRecursiveDownstream,
					ConfigType: "Kubernetes::Pod,Kubernetes::ReplicaSet",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(3)))
				Expect(len(response.Changes)).To(Equal(3))
				Expect(response.Summary["Pulled"]).To(Equal(2))
				Expect(response.Summary["diff"]).To(Equal(1))
			})

			ginkgo.It("NOT IN", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID:  U.ID.String(),
					Recursive:  query.CatalogChangeRecursiveDownstream,
					ConfigType: "!Kubernetes::ReplicaSet",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(5)))
				Expect(len(response.Changes)).To(Equal(5))
				Expect(response.Summary["diff"]).To(Equal(2))
				Expect(response.Summary["Pulled"]).To(Equal(2))
				Expect(response.Summary["RegisterNode"]).To(Equal(1))
			})
		})

		ginkgo.Context("Change type filter", func() {
			ginkgo.It("IN", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID:  X.ID.String(),
					Recursive:  query.CatalogChangeRecursiveAll,
					ChangeType: "diff",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(2)))
				Expect(len(response.Changes)).To(Equal(2))
				Expect(response.Summary["diff"]).To(Equal(2))
			})

			ginkgo.It("NOT IN", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID:  U.ID.String(),
					Recursive:  query.CatalogChangeRecursiveDownstream,
					ChangeType: "!diff,!Pulled",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(1)))
				Expect(len(response.Changes)).To(Equal(1))
				Expect(response.Summary["RegisterNode"]).To(Equal(1))
			})
		})

		ginkgo.Context("Severity filter", func() {
			ginkgo.It("NOT", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					Severity:  "!info",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(3)))
				Expect(len(response.Changes)).To(Equal(3))
				Expect(response.Summary["Pulled"]).To(Equal(1))
				Expect(response.Summary["diff"]).To(Equal(2))
			})

			ginkgo.It("should return the given severity and higher", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					Severity:  string(models.SeverityMedium),
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(2)))
				Expect(len(response.Changes)).To(Equal(2))
				Expect(response.Summary["Pulled"]).To(Equal(1))
				Expect(response.Summary["diff"]).To(Equal(1))
			})
		})

		ginkgo.Context("Pagination", func() {
			ginkgo.It("Page size", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					SortBy:    "summary",
					PageSize:  2,
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(6)))
				Expect(len(response.Changes)).To(Equal(2))
				changes := lo.Map(response.Changes, func(c query.ConfigChangeRow, _ int) string { return c.Summary })
				Expect(changes).To(Equal([]string{".name.U", ".name.V"}))
			})

			ginkgo.It("Page number", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					SortBy:    "summary",
					PageSize:  2,
					Page:      2,
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(6)))
				Expect(len(response.Changes)).To(Equal(2))
				changes := lo.Map(response.Changes, func(c query.ConfigChangeRow, _ int) string { return c.Summary })
				Expect(changes).To(Equal([]string{".name.W", ".name.X"}))
			})
		})

		ginkgo.Context("recursive mode", func() {
			ginkgo.It("upstream", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: W.ID.String(),
					Recursive: query.CatalogChangeRecursiveUpstream,
				})
				Expect(err).To(BeNil())
				Expect(len(response.Changes)).To(Equal(2))
				Expect(response.Total).To(Equal(int64(2)))
				Expect(response.Summary[UChange.ChangeType]).To(Equal(1))
				Expect(response.Summary[WChange.ChangeType]).To(Equal(1))
			})

			ginkgo.It(query.CatalogChangeRecursiveDownstream, func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: V.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
				})
				Expect(err).To(BeNil())
				Expect(len(response.Changes)).To(Equal(4))
				Expect(response.Total).To(Equal(int64(4)))
				Expect(response.Summary["diff"]).To(Equal(3))
				Expect(response.Summary["Pulled"]).To(Equal(1))
			})

			ginkgo.It(query.CatalogChangeRecursiveAll, func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: V.ID.String(),
					Recursive: query.CatalogChangeRecursiveAll,
				})
				Expect(err).To(BeNil())
				Expect(len(response.Changes)).To(Equal(5))
				Expect(response.Total).To(Equal(int64(5)))
				Expect(response.Summary["diff"]).To(Equal(3))
				Expect(response.Summary["Pulled"]).To(Equal(1))
				Expect(response.Summary["RegisterNode"]).To(Equal(1))
			})
		})

		ginkgo.It("should handle datemath", func() {
			response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				CatalogID: U.ID.String(),
				Recursive: query.CatalogChangeRecursiveDownstream,
				From:      "now-65m",
				To:        "now-1s",
			})
			Expect(err).To(BeNil())
			Expect(response.Total).To(Equal(int64(2)))
			Expect(len(response.Changes)).To(Equal(2))
			Expect(response.Summary["diff"]).To(Equal(1))
			Expect(response.Summary["RegisterNode"]).To(Equal(1))
		})

		ginkgo.Context("Sorting", func() {
			ginkgo.It("Descending", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					SortBy:    "-name",
				})
				Expect(err).To(BeNil())
				Expect(len(response.Changes)).To(Equal(6))
				Expect(response.Total).To(Equal(int64(6)))
				changes := lo.Map(response.Changes, func(c query.ConfigChangeRow, _ int) string { return c.ConfigName })
				Expect(changes).To(Equal([]string{"Z", "Y", "X", "W", "V", "U"}))
			})

			ginkgo.It("Ascending", func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: U.ID.String(),
					Recursive: query.CatalogChangeRecursiveDownstream,
					SortBy:    "name",
				})
				Expect(err).To(BeNil())
				Expect(response.Total).To(Equal(int64(6)))
				Expect(len(response.Changes)).To(Equal(6))
				changes := lo.Map(response.Changes, func(c query.ConfigChangeRow, _ int) string { return c.ConfigName })
				Expect(changes).To(Equal([]string{"U", "V", "W", "X", "Y", "Z"}))
			})
		})
	})
})
