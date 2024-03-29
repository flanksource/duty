package tests

import (
	"fmt"

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
			{ConfigID: configItems["helm-release-of-deployment"].ID.String(), RelatedID: configItems["deployment"].ID.String(), Relation: "HelmReleaseDeployment"},
			{ConfigID: configItems["kustomize-of-helm-release"].ID.String(), RelatedID: configItems["helm-release-of-deployment"].ID.String(), Relation: "KustomizationHelmRelease"},
		}
		err = ctx.DB().Clauses(clause.OnConflict{DoNothing: true}).Save(configRelations).Error
		Expect(err).ToNot(HaveOccurred())

		err = query.SyncConfigCache(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		got := query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::HelmRelease", "incoming")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["helm-release-of-deployment"].ID.String(), "Kubernetes::Kustomization", "incoming")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["kustomize-of-helm-release"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::Kustomization", "incoming")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["kustomize-of-helm-release"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::Pod", "incoming")
		Expect(got).To(BeNil())

		got = query.TraverseConfig(DefaultContext, configItems["deployment"].ID.String(), "Kubernetes::Node", "incoming")
		Expect(got).To(BeNil())

		// TODO Fix these
		got = query.TraverseConfig(DefaultContext, configItems["helm-release-of-deployment"].ID.String(), "Kubernetes::Deployment", "outgoing")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["deployment"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["kustomize-of-helm-release"].ID.String(), "Kubernetes::HelmRelease", "outgoing")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		got = query.TraverseConfig(DefaultContext, configItems["kustomize-of-helm-release"].ID.String(), "Kubernetes::Deployment", "outgoing")
		Expect(got).ToNot(BeNil())
		Expect(got[0].ID.String()).To(Equal(configItems["deployment"].ID.String()))

		// Test with CEL Exprs
		templateEnv := map[string]any{
			"configID":          configItems["deployment"].ID.String(),
			"configIDKustomize": configItems["kustomize-of-helm-release"].ID.String(),
		}

		template := gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::HelmRelease', 'incoming')[0].id",
		}
		gotExpr, err := DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotExpr).To(Equal(configItems["helm-release-of-deployment"].ID.String()))

		template = gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::Kustomization', 'incoming')[0].name",
		}
		gotExpr, err = DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotExpr).To(Equal(*configItems["kustomize-of-helm-release"].Name))

		template = gomplate.Template{
			Expression: "catalog.traverse(configID, 'Kubernetes::Pod', 'incoming')[0].name",
		}
		gotExpr, err = DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).To(HaveOccurred())
		Expect(gotExpr).To(Equal(""))

		template = gomplate.Template{
			Expression: "catalog.traverse(configIDKustomize, 'Kubernetes::Deployment', 'outgoing')[0].name",
		}
		gotExpr, err = DefaultContext.RunTemplate(template, templateEnv)
		Expect(err).ToNot(HaveOccurred())
		Expect(gotExpr).To(Equal(*configItems["deployment"].Name))

		// Testing struct templater
		t := DefaultContext.NewStructTemplater(map[string]any{"id": configItems["deployment"].ID.String()}, "", nil)
		inlineStruct := struct {
			Name string
			Type string
		}{
			Name: "Name is {{ (index (catalog_traverse .id  \"Kubernetes::Kustomization\" \"incoming\") 0).Name }}",
			Type: "Type is {{ (index (catalog_traverse .id  \"Kubernetes::Kustomization\" \"incoming\") 0).Type }}",
		}

		err = t.Walk(&inlineStruct)
		Expect(err).ToNot(HaveOccurred())
		Expect(inlineStruct.Name).To(Equal(fmt.Sprintf("Name is %s", *configItems["kustomize-of-helm-release"].Name)))
		Expect(inlineStruct.Type).To(Equal(fmt.Sprintf("Type is %s", *configItems["kustomize-of-helm-release"].Type)))

		// Cleanup for normal tests to pass
		err = ctx.DB().Where("config_id in ?", lo.Map(lo.Values(configItems), func(c models.ConfigItem, _ int) string { return c.ID.String() })).Delete(&models.ConfigRelationship{}).Error
		Expect(err).ToNot(HaveOccurred())

		err = ctx.DB().Delete(lo.Values(configItems)).Error
		Expect(err).ToNot(HaveOccurred())
	})

})
