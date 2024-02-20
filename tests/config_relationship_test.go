package tests

import (
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

	ginkgo.It("should return INCOOMING relationships", func() {
		var relatedConfigs []models.RelatedConfig
		err := DefaultContext.DB().Raw("SELECT * FROM related_configs(?, 'all', false)", dummy.KubernetesNodeA.ID).Find(&relatedConfigs).Error
		Expect(err).To(BeNil())

		Expect(len(relatedConfigs)).To(Equal(1))
		Expect(relatedConfigs[0].Relation).To(Equal("ClusterNode"))
		Expect(relatedConfigs[0].Type).To(Equal(models.RelatedConfigTypeIncoming))
		Expect(relatedConfigs[0].Config["id"]).To(Equal(dummy.KubernetesCluster.ID.String()))
	})
})
