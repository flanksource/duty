package tests

import (
	"github.com/flanksource/duty/models"
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
		U = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("U")}
		V = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("V")}
		W = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("W")}
		X = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("X")}
		Y = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("Y")}
		Z = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-changes"), Name: lo.ToPtr("Z")}
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
		UChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: U.ID.String(), Summary: ".name.U", Source: "test-changes"}
		VChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: V.ID.String(), Summary: ".name.V", Source: "test-changes"}
		WChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: W.ID.String(), Summary: ".name.W", Source: "test-changes"}
		XChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: X.ID.String(), Summary: ".name.X", Source: "test-changes"}
		YChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: Y.ID.String(), Summary: ".name.Y", Source: "test-changes"}
		ZChange = models.ConfigChange{ID: uuid.New().String(), ConfigID: Z.ID.String(), Summary: ".name.Z", Source: "test-changes"}
	)
	changes := []models.ConfigChange{UChange, VChange, WChange, XChange, YChange, ZChange}

	ginkgo.BeforeAll(func() {
		// Save configs
		err := DefaultContext.DB().Create(&configItems).Error
		Expect(err).To(BeNil())

		var foundConfigs []models.ConfigItem
		err = DefaultContext.DB().Select("id").Where("namespace = 'test-changes'").Find(&foundConfigs).Error
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

		err = DefaultContext.DB().Where("namespace = 'test-changes'").Delete(&models.ConfigItem{}).Error
		Expect(err).To(BeNil())
	})

	ginkgo.Context("Downstream", func() {
		ginkgo.It("should return changes of a root node", func() {
			var relatedChanges []models.ConfigChange
			err := DefaultContext.DB().Raw("SELECT * FROM related_changes_recursive(?)", U.ID).Find(&relatedChanges).Error
			Expect(err).To(BeNil())

			Expect(len(relatedChanges)).To(Equal(6))

			relatedIDs := lo.Map(relatedChanges, func(rc models.ConfigChange, _ int) string { return rc.ID })
			Expect(relatedIDs).To(HaveExactElements([]string{UChange.ID, VChange.ID, WChange.ID, XChange.ID, YChange.ID, ZChange.ID}))
		})

		ginkgo.It("should return changes of a leaf node", func() {
			var relatedChanges []models.ConfigChange
			err := DefaultContext.DB().Raw("SELECT * FROM related_changes_recursive(?)", Z.ID).Find(&relatedChanges).Error
			Expect(err).To(BeNil())

			Expect(len(relatedChanges)).To(Equal(1))

			relatedIDs := lo.Map(relatedChanges, func(rc models.ConfigChange, _ int) string { return rc.ID })
			Expect(relatedIDs).To(HaveExactElements([]string{ZChange.ID}))
		})
	})

	ginkgo.Context("Upstream", func() {
		ginkgo.It("should return changes for a root node", func() {
			var relatedChanges []models.ConfigChange
			err := DefaultContext.DB().Raw("SELECT * FROM related_changes_recursive(?, 'upstream')", U.ID).Find(&relatedChanges).Error
			Expect(err).To(BeNil())

			Expect(len(relatedChanges)).To(Equal(1))

			relatedIDs := lo.Map(relatedChanges, func(rc models.ConfigChange, _ int) string { return rc.ID })
			Expect(relatedIDs).To(HaveExactElements([]string{UChange.ID}))
		})

		ginkgo.It("should return changes of a non-root node", func() {
			var relatedChanges []models.ConfigChange
			err := DefaultContext.DB().Raw("SELECT * FROM related_changes_recursive(?, 'upstream')", X.ID).Find(&relatedChanges).Error
			Expect(err).To(BeNil())

			Expect(len(relatedChanges)).To(Equal(3))

			relatedIDs := lo.Map(relatedChanges, func(rc models.ConfigChange, _ int) string { return rc.ID })
			Expect(relatedIDs).To(HaveExactElements([]string{UChange.ID, VChange.ID, XChange.ID}))
		})
	})
})
