package tests

import (
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

var _ = ginkgo.Describe("Config traversal", ginkgo.Ordered, func() {
	ginkgo.It("should be able to traverse config relationships via types", func() {
		configItems := map[string]models.ConfigItem{
			"deployment":                 {ID: uuid.New(), Name: utils.Ptr("canary-checker"), Type: utils.Ptr("Kubernetes::Deployment")},
			"helm-release-of-deployment": {ID: uuid.New(), Name: utils.Ptr("mission-control"), Type: utils.Ptr("Kubernetes::HelmRelease")},
			"kustomize-of-helm-release":  {ID: uuid.New(), Name: utils.Ptr("aws-demo-infra"), Type: utils.Ptr("Kubernetes::Kustomization")},
		}
		ctx := DefaultContext
		err := ctx.DB().Save(lo.Values(configItems)).Error
		Expect(err).ToNot(HaveOccurred())

		configRelations := []models.ConfigRelationship{
			{ConfigID: configItems["deployment"].ID.String(), RelatedID: configItems["helm-release-of-deployment"].ID.String(), Relation: "HelmReleaseDeployment"},
			{ConfigID: configItems["helm-release-of-deployment"].ID.String(), RelatedID: configItems["kustomize-of-helm-release"].ID.String(), Relation: "KustomizationHelmRelease"},
		}
		err = ctx.DB().Clauses(clause.OnConflict{DoNothing: true}).Save(configRelations).Error
		Expect(err).ToNot(HaveOccurred())

		err = query.SyncConfigCache(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		got, err := query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.ID.String()).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		got, err = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease/Kubernetes::Kustomization")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.ID.String()).To(Equal(configItems["kustomize-of-helm-release"].ID.String()))

		_, err = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::Pod")
		Expect(err).To(HaveOccurred())

		_, err = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease/Kubernetes::Node")
		Expect(err).To(HaveOccurred())

		// Cleanup for normal tests to pass
		err = ctx.DB().Where("config_id in ?", lo.Map(lo.Values(configItems), func(c models.ConfigItem, _ int) string { return c.ID.String() })).Delete(&models.ConfigRelationship{}).Error
		Expect(err).ToNot(HaveOccurred())

		err = ctx.DB().Delete(lo.Values(configItems)).Error
		Expect(err).ToNot(HaveOccurred())
	})

})
