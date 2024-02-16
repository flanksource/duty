package tests

import (
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/gomplate/v3"
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

		got := query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease")
		Expect(got).ToNot(BeNil())
		Expect(got.ID.String()).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease/Kubernetes::Kustomization")
		Expect(got).ToNot(BeNil())
		Expect(got.ID.String()).To(Equal(configItems["kustomize-of-helm-release"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::Pod")
		Expect(got).To(BeNil())

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease/Kubernetes::Node")
		Expect(got).To(BeNil())

		// Test with CEL Exprs
		templateEnv := map[string]any{
			"configID": configItems["deployment"].ID.String(),
		}

		template := gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::HelmRelease').id",
		}
		gotExpr, err := DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotExpr).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		template = gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::HelmRelease/Kubernetes::Kustomization').name",
		}
		gotExpr, err = DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotExpr).To(Equal(*configItems["kustomize-of-helm-release"].Name))

		template = gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::Pod').name",
		}
		gotExpr, err = DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).To(HaveOccurred())
		Expect(gotExpr).To(Equal(""))

		// Cleanup for normal tests to pass
		err = ctx.DB().Where("config_id in ?", lo.Map(lo.Values(configItems), func(c models.ConfigItem, _ int) string { return c.ID.String() })).Delete(&models.ConfigRelationship{}).Error
		Expect(err).ToNot(HaveOccurred())

		err = ctx.DB().Delete(lo.Values(configItems)).Error
		Expect(err).ToNot(HaveOccurred())
	})

})
