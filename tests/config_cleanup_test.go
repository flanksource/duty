package tests

import (
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Config cleanup", ginkgo.Ordered, func() {

	generator := ConfigGenerator{
		Nodes:                  10,
		PodPerDeployment:       3,
		DeploymentPerNamespace: 5,
		DeletedPercentage:      20,
		Namespaces:             3,
		NumChangesPerConfig:    2,
		NumInsightsPerConfig:   2,
		UnhealthyPercentage:    10,
		UnknownPercentage:      1,
	}

	ginkgo.BeforeAll(func() {
		generator.GenerateKubernetes()

		Expect(generator.Save(DefaultContext.DB())).To(BeNil())

	})

	ginkgo.It("should cleanup deleted items", func() {

		deleted, err := job.DeleteOldConfigItems(DefaultContext, 3)
		Expect(err).To(BeNil())
		Expect(deleted).To(BeNumerically("==", lo.CountBy(generator.Generated.Configs,
			func(item models.ConfigItem) bool {
				return item.DeletedAt != nil
			})))
	})

})
