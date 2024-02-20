package tests

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Config relationship recursive", ginkgo.Ordered, ginkgo.Focus, func() {
	//       A
	//      / \
	//     B   C
	//    / \   \
	//   D   E   F
	//  / \
	// G   H
	//    /
	//   A

	// Create a list of ConfigItems
	var (
		A = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("A")}
		B = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("B")}
		C = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("C")}
		D = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("D")}
		E = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("E")}
		F = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("F")}
		G = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("G")}
		H = models.ConfigItem{ID: uuid.New(), Namespace: lo.ToPtr("test-relationship"), Name: lo.ToPtr("H")}
	)
	configItems := []models.ConfigItem{A, B, C, D, E, F, G, H}

	// Create relationships between ConfigItems
	relationships := []models.ConfigRelationship{
		{ConfigID: A.ID.String(), RelatedID: B.ID.String(), Relation: "test-relationship-AB"},
		{ConfigID: A.ID.String(), RelatedID: C.ID.String(), Relation: "test-relationship-AC"},
		{ConfigID: B.ID.String(), RelatedID: D.ID.String(), Relation: "test-relationship-BD"},
		{ConfigID: B.ID.String(), RelatedID: E.ID.String(), Relation: "test-relationship-BE"},
		{ConfigID: C.ID.String(), RelatedID: F.ID.String(), Relation: "test-relationship-CF"},
		{ConfigID: D.ID.String(), RelatedID: G.ID.String(), Relation: "test-relationship-DG"},
		{ConfigID: D.ID.String(), RelatedID: H.ID.String(), Relation: "test-relationship-DH"},
		{ConfigID: H.ID.String(), RelatedID: A.ID.String(), Relation: "test-relationship-HA"},
	}

	ginkgo.BeforeAll(func() {
		err := DefaultContext.DB().Create(&configItems).Error
		Expect(err).To(BeNil())

		var foundConfigs []models.ConfigItem
		err = DefaultContext.DB().Select("id").Where("namespace = 'test-relationship'").Find(&foundConfigs).Error
		Expect(err).To(BeNil())
		Expect(len(foundConfigs)).To(Equal(len(configItems)))

		err = DefaultContext.DB().Create(&relationships).Error
		Expect(err).To(BeNil())

		var foundRelationships []models.ConfigRelationship
		err = DefaultContext.DB().Where("relation LIKE 'test-relationship%'").Find(&foundRelationships).Error
		Expect(err).To(BeNil())
		Expect(len(foundRelationships)).To(Equal(len(relationships)))
	})

	ginkgo.It("should return OUTGOING relationships", func() {
		var relatedConfigs []models.RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?)", A.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		for _, rc := range relatedConfigs {
			fmt.Println(rc.Level, strings.TrimPrefix(rc.Relation, "test-relationship-"))
		}

		Expect(len(relatedConfigs)).To(Equal(7))
	})

	ginkgo.It("should return INCOMING relationships", func() {
		var relatedConfigs []models.RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs_recursive(?, 'incoming', false)", F.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(2))
	})
})

var _ = ginkgo.Describe("Config relationship", ginkgo.Ordered, func() {
	ginkgo.It("should return OUTGOING relationships", func() {
		var relatedConfigs []models.RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs(?)", dummy.KubernetesCluster.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(2))
		for _, rc := range relatedConfigs {
			Expect(rc.Relation).To(Equal("ClusterNode"))
			Expect(rc.Type).To(Equal(models.RelatedConfigTypeOutgoing))
			Expect(rc.Config["id"]).To(BeElementOf([]string{dummy.KubernetesNodeA.ID.String(), dummy.KubernetesNodeB.ID.String()}))
		}
	})

	ginkgo.It("should return INCOMING relationships", func() {
		var relatedConfigs []models.RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs(?, 'all', false)", dummy.KubernetesNodeA.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(1))
		Expect(relatedConfigs[0].Relation).To(Equal("ClusterNode"))
		Expect(relatedConfigs[0].Type).To(Equal(models.RelatedConfigTypeIncoming))
		Expect(relatedConfigs[0].Config["id"]).To(Equal(dummy.KubernetesCluster.ID.String()))
	})
})
