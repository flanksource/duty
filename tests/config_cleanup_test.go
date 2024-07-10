package tests

import (
	"github.com/flanksource/duty/models"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		Tags: map[string]string{
			"test": "true",
		},
	}

	ginkgo.BeforeAll(func() {
		generator.GenerateKubernetes()
		Expect(generator.Save(DefaultContext.DB())).To(BeNil())
	})

	ginkgo.Context("deployments", func() {
		ginkgo.It("should have created the deployments", func() {
			assertConfigCount("Kubernetes::Deployment", DeleteFilterNone, generator.DeploymentPerNamespace*generator.Namespaces)
		})

		ginkgo.It("should have the correct number of deleted deployments", func() {
			expectedDeleted := float32(float32(generator.DeletedPercentage)/100) * float32(generator.DeploymentPerNamespace*generator.Namespaces)
			assertConfigCount("Kubernetes::Deployment", DeleteFilterDeleted, int(expectedDeleted))
		})

		ginkgo.It("should have the correct number of active deployments", func() {
			expectedDeleted := float32(float32(100-generator.DeletedPercentage)/100) * float32(generator.DeploymentPerNamespace*generator.Namespaces)
			assertConfigCount("Kubernetes::Deployment", DeleteFilterNotDeleted, int(expectedDeleted))
		})
	})
})

func assertConfigCount(itemType string, deleteFilter DeleteFilter, expected int) {
	count, err := getConfigItemCount(itemType, deleteFilter)
	Expect(err).To(BeNil())
	Expect(count).To(Equal(expected))
}

func getConfigItemCount(itemType string, deleteFilter DeleteFilter) (int, error) {
	var count int
	query := DefaultContext.DB().Model(&models.ConfigItem{}).
		Select("COUNT(*)").
		Where("type = ?", itemType).
		Where("tags->>'test' = ?", "true")

	switch deleteFilter {
	case DeleteFilterDeleted:
		query = query.Where("deleted_at IS NOT NULL")
	case DeleteFilterNotDeleted:
		query = query.Where("deleted_at IS NULL")
	}

	err := query.Find(&count).Error
	return count, err
}

type DeleteFilter int

const (
	DeleteFilterNone DeleteFilter = iota
	DeleteFilterDeleted
	DeleteFilterNotDeleted
)
