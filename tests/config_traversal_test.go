package tests

import (
	"fmt"

	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/gomplate/v3"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
)

func assertTraverseConfig(from models.ConfigItem, relationType string, direction string, to ...models.ConfigItem) {
	got := query.TraverseConfig(DefaultContext, from.ID.String(), relationType, direction)
	Expect(got).To(EqualConfigs(to...))
}

func traverseTemplate(from models.ConfigItem, relationType string, direction string) string {
	templateEnv := map[string]any{
		"configID": from.ID.String(),
	}

	template := gomplate.Template{
		Expression: fmt.Sprintf("dyn(catalog.traverse(configID, '%s', '%s')).map(i, i.id).join(' ')", relationType, direction),
	}
	gotExpr, err := DefaultContext.RunTemplate(template, templateEnv)
	Expect(err).ToNot(HaveOccurred())
	return gotExpr
}

var _ = ginkgo.Describe("Config traversal", ginkgo.Ordered, func() {
	ginkgo.It("should be able to traverse config relationships via types", func() {
		deployment := models.ConfigItem{ID: uuid.New(), Name: utils.Ptr("canary-checker"), Type: utils.Ptr("Kubernetes::Deployment"), ConfigClass: "Deployment"}
		helmRelease := models.ConfigItem{ID: uuid.New(), Name: utils.Ptr("mission-control"), Type: utils.Ptr("Kubernetes::HelmRelease"), ConfigClass: "HelmRelease"}
		kustomize := models.ConfigItem{ID: uuid.New(), Name: utils.Ptr("aws-demo-infra"), Type: utils.Ptr("Kubernetes::Kustomization"), ConfigClass: "Kustomization"}
		bootstrap := models.ConfigItem{ID: uuid.New(), Name: utils.Ptr("aws-demo-bootstrap"), Type: utils.Ptr("Kubernetes::Kustomization"), ConfigClass: "Kustomization"}
		all := []models.ConfigItem{deployment, helmRelease, kustomize, bootstrap}
		ctx := DefaultContext
		err := ctx.DB().Save(all).Error
		Expect(err).ToNot(HaveOccurred())

		configRelations := []models.ConfigRelationship{
			{ConfigID: helmRelease.ID.String(), RelatedID: deployment.ID.String(), Relation: "HelmReleaseDeployment"},
			{ConfigID: kustomize.ID.String(), RelatedID: helmRelease.ID.String(), Relation: "KustomizationHelmRelease"},
			{ConfigID: bootstrap.ID.String(), RelatedID: kustomize.ID.String(), Relation: "KustomizationKustomization"},
		}
		err = ctx.DB().Clauses(clause.OnConflict{DoNothing: true}).Save(configRelations).Error
		Expect(err).ToNot(HaveOccurred())

		err = job.RefreshConfigItemSummary7d(DefaultContext)
		Expect(err).To(BeNil())

		err = query.SyncConfigCache(DefaultContext)
		Expect(err).ToNot(HaveOccurred())

		assertTraverseConfig(deployment, "Kubernetes::HelmRelease", "incoming", helmRelease)

		assertTraverseConfig(helmRelease, "Kubernetes::Kustomization", "incoming", kustomize, bootstrap)

		assertTraverseConfig(deployment, "Kubernetes::Kustomization", "incoming", bootstrap, kustomize)

		assertTraverseConfig(deployment, "Kubernetes::HelmRelease/Kubernetes::Kustomization", "incoming", kustomize, bootstrap)

		got := query.TraverseConfig(DefaultContext, deployment.ID.String(), "Kubernetes::Kustomization/Kubernetes::Kustomization", "incoming")
		Expect(got).ToNot(BeNil())
		// This should only return 1 object since we are
		// passing explicit path for the boostrap kustomization
		Expect(len(got)).To(Equal(1))
		Expect(got[0].ID.String()).To(Equal(bootstrap.ID.String()))

		assertTraverseConfig(deployment, "Kubernetes::Pod", "incoming")

		assertTraverseConfig(deployment, "Kubernetes::Node", "incoming")
		assertTraverseConfig(deployment, "Kubernetes::HelmRelease/Kubernetes::Node", "incoming")

		assertTraverseConfig(helmRelease, "Kubernetes::Deployment", "outgoing", deployment)

		assertTraverseConfig(kustomize, "Kubernetes::HelmRelease", "outgoing", helmRelease)

		assertTraverseConfig(kustomize, "Kubernetes::Deployment", "outgoing", deployment)

		Expect(traverseTemplate(deployment, "Kubernetes::HelmRelease", "incoming")).
			To(Equal(helmRelease.ID.String()))

		Expect(traverseTemplate(deployment, "Kubernetes::Kustomization", "incoming")).
			To(Equal(fmt.Sprintf("%s %s", kustomize.ID, bootstrap.ID)))

		Expect(traverseTemplate(deployment, "Kubernetes::Pod", "incoming")).
			To(BeEmpty())

		Expect(traverseTemplate(kustomize, "Kubernetes::Deployment", "outgoing")).
			To(Equal(deployment.ID.String()))

		// Testing struct templater
		t := DefaultContext.NewStructTemplater(map[string]any{"id": deployment.ID.String()}, "", nil)
		inlineStruct := struct {
			Name string
			Type string
		}{
			Name: "Name is {{ (index (catalog_traverse .id  \"Kubernetes::Kustomization\" \"incoming\") 0).Name }}",
			Type: "Type is {{ (index (catalog_traverse .id  \"Kubernetes::Kustomization\" \"incoming\") 0).Type }}",
		}

		err = t.Walk(&inlineStruct)
		Expect(err).ToNot(HaveOccurred())
		Expect(inlineStruct.Name).To(Equal(fmt.Sprintf("Name is %s", *kustomize.Name)))
		Expect(inlineStruct.Type).To(Equal(fmt.Sprintf("Type is %s", *kustomize.Type)))

		// Cleanup for normal tests to pass
		err = ctx.DB().Where("config_id in ?", lo.Map(all, func(c models.ConfigItem, _ int) string { return c.ID.String() })).Delete(&models.ConfigRelationship{}).Error
		Expect(err).ToNot(HaveOccurred())

		err = ctx.DB().Delete(all).Error
		Expect(err).ToNot(HaveOccurred())
	})

})
