package tests

import (
	"strings"
	"time"

	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/db"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

var _ = ginkgo.Describe("Config changes recursive", ginkgo.Ordered, func() {
	// Graph #1 (acyclic)
	//
	//          U --- (A)
	//         / \
	// (B)----V   W
	//       / \
	// (C)--X   Y
	//     /
	//    Z--- (D)

	// Create a list of ConfigItems
	var (
		U = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Node"), Name: lo.ToPtr("U"), ConfigClass: "Node"}
		V = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Deployment"), Name: lo.ToPtr("V"), ConfigClass: "Deployment", ParentID: lo.ToPtr(U.ID), Path: U.ID.String()}
		W = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Pod"), Name: lo.ToPtr("W"), ConfigClass: "Pod", ParentID: lo.ToPtr(U.ID), Path: U.ID.String()}
		X = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ReplicaSet"), Name: lo.ToPtr("X"), ConfigClass: "ReplicaSet", ParentID: lo.ToPtr(V.ID), Path: strings.Join([]string{U.ID.String(), V.ID.String()}, ".")}
		Y = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::PersistentVolume"), Name: lo.ToPtr("Y"), ConfigClass: "PersistentVolume", ParentID: lo.ToPtr(V.ID), Path: strings.Join([]string{U.ID.String(), V.ID.String()}, ".")}
		Z = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::Pod"), Name: lo.ToPtr("Z"), ConfigClass: "Pod", ParentID: lo.ToPtr(X.ID), Path: strings.Join([]string{U.ID.String(), V.ID.String(), X.ID.String()}, ".")}

		A = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ConfigMap"), Name: lo.ToPtr("A"), ConfigClass: "ConfigMap"}
		B = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ConfigMap"), Name: lo.ToPtr("B"), ConfigClass: "ConfigMap"}
		C = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ConfigMap"), Name: lo.ToPtr("C"), ConfigClass: "ConfigMap"}
		D = models.ConfigItem{ID: uuid.New(), Tags: types.JSONStringMap{"namespace": "test-changes"}, Type: lo.ToPtr("Kubernetes::ConfigMap"), Name: lo.ToPtr("D"), ConfigClass: "ConfigMap"}
	)
	configItems := []models.ConfigItem{U, V, W, X, Y, Z, A, B, C, D}

	// Create relationships between ConfigItems
	relationships := []models.ConfigRelationship{
		{ConfigID: U.ID.String(), RelatedID: A.ID.String(), Relation: "test-changes-UA"},
		{ConfigID: V.ID.String(), RelatedID: B.ID.String(), Relation: "test-changes-VB"},
		{ConfigID: X.ID.String(), RelatedID: C.ID.String(), Relation: "test-changes-XC"},
		{ConfigID: Z.ID.String(), RelatedID: D.ID.String(), Relation: "test-changes-ZD"},
	}

	// Create changes for each config
	var (
		UChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now()), Severity: "info", ConfigID: U.ID.String(), Summary: ".name.U", ChangeType: "RegisterNode", Source: "test-changes"}
		VChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour)), Severity: models.SeverityHigh, ConfigID: V.ID.String(), Summary: ".name.V", ChangeType: "diff", Source: "test-changes"}
		WChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 2)), Severity: models.SeverityCritical, ConfigID: W.ID.String(), Summary: ".name.W", ChangeType: "Pulled", Source: "test-changes"}
		XChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 3)), Severity: "info", ConfigID: X.ID.String(), Summary: ".name.X", ChangeType: "diff", Source: "test-changes"}
		YChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 4)), Severity: "warn", ConfigID: Y.ID.String(), Summary: ".name.Y", ChangeType: "diff", Source: "test-changes"}
		ZChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: Z.ID.String(), Summary: ".name.Z", ChangeType: "Pulled", Source: "test-changes"}

		AChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: A.ID.String(), Summary: ".name.A", ChangeType: "Pulled", Source: "test-changes"}
		BChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: B.ID.String(), Summary: ".name.B", ChangeType: "Pulled", Source: "test-changes"}
		CChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: C.ID.String(), Summary: ".name.C", ChangeType: "Pulled", Source: "test-changes"}
		DChange = models.ConfigChange{ID: uuid.New().String(), CreatedAt: lo.ToPtr(time.Now().Add(-time.Hour * 5)), Severity: "info", ConfigID: D.ID.String(), Summary: ".name.D", ChangeType: "Pulled", Source: "test-changes"}

		changes = []models.ConfigChange{UChange, VChange, WChange, XChange, YChange, ZChange, AChange, BChange, CChange, DChange}
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

	var findChanges = func(id uuid.UUID, filter query.ChangeRelationDirection, deleted bool) (*query.CatalogChangesSearchResponse, error) {
		return query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
			CatalogID:             id.String(),
			IncludeDeletedConfigs: deleted,
			Recursive:             filter,
		})
	}

	ginkgo.Context("Both ways", func() {
		ginkgo.It("should return changes upstream and downstream", func() {
			relatedChanges, err := findChanges(X.ID, "all", false)
			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(6))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, XChange.ID, ZChange.ID, CChange.ID, DChange.ID}))
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

		ginkgo.It("should return changes of a root node along with soft", func() {
			relatedChanges, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				CatalogID: U.ID.String(),
				Recursive: "downstream",
				Soft:      true,
			})

			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(7))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, WChange.ID, XChange.ID, YChange.ID, ZChange.ID, AChange.ID}))
		})

		ginkgo.It("should return changes of a leaf node", func() {
			relatedChanges, err := findChanges(Z.ID, "all", false)
			Expect(err).To(BeNil())
			Expect(len(relatedChanges.Changes)).To(Equal(5))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{ZChange.ID, XChange.ID, VChange.ID, UChange.ID, DChange.ID}))
		})

		ginkgo.It("should return changes of a leaf node along with soft", func() {
			relatedChanges, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				CatalogID: X.ID.String(),
				Recursive: "downstream",
				Soft:      true,
			})

			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(3))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{XChange.ID, ZChange.ID, CChange.ID}))
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

		ginkgo.It("should return changes of a leaf node along with soft", func() {
			relatedChanges, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				CatalogID: U.ID.String(),
				Recursive: "upstream",
				Soft:      true,
			})

			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(2))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, AChange.ID}))
		})

		ginkgo.It("should return changes of a non-root node", func() {
			relatedChanges, err := findChanges(X.ID, "all", false)
			Expect(err).To(BeNil())
			Expect(len(relatedChanges.Changes)).To(Equal(6))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{UChange.ID, VChange.ID, ZChange.ID, XChange.ID, CChange.ID, DChange.ID}))
		})

		ginkgo.It("should return changes of a leaf node along with soft", func() {
			relatedChanges, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				CatalogID: X.ID.String(),
				Recursive: "upstream",
				Soft:      true,
			})

			Expect(err).To(BeNil())

			Expect(len(relatedChanges.Changes)).To(Equal(4))

			relatedIDs := lo.Map(relatedChanges.Changes, func(rc query.ConfigChangeRow, _ int) string { return rc.ID })
			Expect(relatedIDs).To(ConsistOf([]string{XChange.ID, UChange.ID, VChange.ID, CChange.ID}))
		})
	})

	ginkgo.Context("FindCatalogChanges func", func() {
		ginkgo.It("Without catalog id", func() {
			response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
				ConfigType: "Kubernetes::Pod,Kubernetes::ReplicaSet",
				ChangeType: "!NotificationSent",
			})
			Expect(err).To(BeNil())

			Expect(response.Total).To(Equal(int64(8)))
			Expect(len(response.Changes)).To(Equal(8))
			Expect(response.Summary["Healthy"]).To(Equal(5))
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

			ginkgo.It(string(query.CatalogChangeRecursiveDownstream), func() {
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

			ginkgo.It(string(query.CatalogChangeRecursiveAll), func() {
				response, err := query.FindCatalogChanges(DefaultContext, query.CatalogChangesSearchRequest{
					CatalogID: V.ID.String(),
					Recursive: query.CatalogChangeRecursiveAll,
				})
				Expect(err).To(BeNil())
				Expect(len(response.Changes)).To(Equal(8))
				Expect(response.Total).To(Equal(int64(8)))
				Expect(response.Summary["diff"]).To(Equal(3))
				Expect(response.Summary["Pulled"]).To(Equal(4))
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
			Expect(response.Total).To(BeNumerically(">=", 1))
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

var _ = ginkgo.Describe("handle external id conflict on config change inserts", ginkgo.Ordered, func() {
	// An arbitrarily chosen time of the event we will be inserting multiple times
	var referenceTime = time.Date(2020, 01, 15, 12, 00, 00, 00, time.UTC)

	dummyChanges := []models.ConfigChange{
		{ConfigID: dummy.LogisticsAPIDeployment.ID.String(), ExternalChangeID: lo.ToPtr("conflict_test_1"), Count: 1, CreatedAt: lo.ToPtr(referenceTime.Add(-time.Minute * 5)), Details: []byte(`{"replicas": "1"}`)},
		{ConfigID: dummy.LogisticsAPIDeployment.ID.String(), ExternalChangeID: lo.ToPtr("conflict_test_2"), Count: 1, CreatedAt: lo.ToPtr(referenceTime.Add(-time.Minute * 4)), Details: []byte(`{"replicas": "2"}`)},
	}

	ginkgo.BeforeAll(func() {
		err := DefaultContext.DB().Create(dummyChanges).Error
		Expect(err).To(BeNil())
	})

	ginkgo.AfterAll(func() {
		err := DefaultContext.DB().Delete(dummyChanges).Error
		Expect(err).To(BeNil())
	})

	ginkgo.It("should increase count when the details is changed", func() {
		c := models.ConfigChange{ConfigID: dummy.LogisticsAPIDeployment.ID.String(), ExternalChangeID: lo.ToPtr("conflict_test_1"), Count: 1, CreatedAt: lo.ToPtr(referenceTime), Details: []byte(`{"replicas": "3"}`)}
		err := DefaultContext.DB().Create(&c).Error
		Expect(err).To(BeNil())

		{
			var inserted models.ConfigChange
			err := DefaultContext.DB().Where("external_change_id = ? AND config_id = ?", c.ExternalChangeID, c.ConfigID).First(&inserted).Error
			Expect(db.ErrorDetails(err)).NotTo(HaveOccurred())
			Expect(inserted.Count).To(Equal(2))
		}
	})

	ginkgo.It("should NOT increase count", func() {
		// insert the same change with the same details and external change id
		// and ensure the count doesn't change.
		for i := 0; i < 10; i++ {
			c := models.ConfigChange{ConfigID: dummy.LogisticsAPIDeployment.ID.String(), ExternalChangeID: lo.ToPtr("conflict_test_1"), CreatedAt: lo.ToPtr(referenceTime), Count: 1, Details: []byte(`{"replicas": "3"}`)}
			err := DefaultContext.DB().Create(&c).Error
			Expect(err).To(BeNil())

			{
				var inserted models.ConfigChange
				err := DefaultContext.DB().Where("external_change_id = ? AND config_id = ?", c.ExternalChangeID, c.ConfigID).First(&inserted).Error
				Expect(db.ErrorDetails(err)).NotTo(HaveOccurred())
				Expect(inserted.Count).To(Equal(2))
			}
		}
	})

	ginkgo.It("should increase count when the details is the same but the created_at is changed", func() {
		c := models.ConfigChange{ConfigID: dummy.LogisticsAPIDeployment.ID.String(), ExternalChangeID: lo.ToPtr("conflict_test_1"), Count: 1, CreatedAt: lo.ToPtr(referenceTime.Add(time.Minute)), Details: []byte(`{"replicas": "3"}`)}
		err := DefaultContext.DB().Create(&c).Error
		Expect(err).To(BeNil())

		{
			var inserted models.ConfigChange
			err := DefaultContext.DB().Where("external_change_id = ? AND config_id = ?", c.ExternalChangeID, c.ConfigID).First(&inserted).Error
			Expect(db.ErrorDetails(err)).NotTo(HaveOccurred())
			Expect(inserted.Count).To(Equal(3))
		}
	})
})
