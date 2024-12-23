package tests

import (
	"github.com/flanksource/duty/models"
	pkgGenerator "github.com/flanksource/duty/tests/generator"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Config Generator", ginkgo.Ordered, func() {
	generator := pkgGenerator.ConfigGenerator{
		Nodes: pkgGenerator.ConfigTypeRequirements{
			Count: 3,
		},
		Namespaces: pkgGenerator.ConfigTypeRequirements{
			Count: 2,
		},
		DeploymentPerNamespace: pkgGenerator.ConfigTypeRequirements{
			Count: 1,
		},
		ReplicaSetPerDeployment: pkgGenerator.ConfigTypeRequirements{
			Count:   4,
			Deleted: 3,
		},
		PodsPerReplicaSet: pkgGenerator.ConfigTypeRequirements{
			Count:                1,
			NumChangesPerConfig:  5,
			NumInsightsPerConfig: 2,
		},
		Tags: map[string]string{
			"test": "true",
		},
	}

	ginkgo.BeforeAll(func() {
		generator.GenerateKubernetes()
		Expect(generator.Save(DefaultContext.DB())).To(BeNil())
	})

	ginkgo.AfterAll(func() {
		err := generator.Destroy(DefaultContext.DB())
		Expect(err).To(BeNil())

		assertConfigCount("", DeleteFilterNone, 0)
	})

	ginkgo.It("should have created namespaces", func() {
		assertConfigCount("Kubernetes::Namespace", DeleteFilterNone, generator.Namespaces.Count)
	})

	ginkgo.Context("deployments", func() {
		ginkgo.It("should have created the deployments", func() {
			assertConfigCount("Kubernetes::Deployment", DeleteFilterNone, generator.DeploymentPerNamespace.Count*generator.Namespaces.Count)
		})

		ginkgo.It("should have the correct number of deleted deployments", func() {
			assertConfigCount("Kubernetes::Deployment", DeleteFilterDeleted, generator.DeploymentPerNamespace.Deleted*generator.Namespaces.Count)
		})

		ginkgo.It("should have the correct number of active deployments", func() {
			assertConfigCount("Kubernetes::Deployment", DeleteFilterNotDeleted, (generator.DeploymentPerNamespace.Count-generator.DeploymentPerNamespace.Deleted)*generator.Namespaces.Count)
		})
	})

	ginkgo.Context("replicasets", func() {
		ginkgo.It("should have created replicasets", func() {
			assertConfigCount("Kubernetes::ReplicaSet", DeleteFilterNone, generator.ReplicaSetPerDeployment.Count*generator.DeploymentPerNamespace.Count*generator.Namespaces.Count)
		})

		ginkgo.It("should have the correct number of deleted replicasets", func() {
			assertConfigCount("Kubernetes::ReplicaSet", DeleteFilterDeleted, generator.ReplicaSetPerDeployment.Deleted*generator.DeploymentPerNamespace.Count*generator.Namespaces.Count)
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
		Where("tags->>'test' = ?", "true")

	if itemType != "" {
		query = query.Where("type = ?", itemType)
	}

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
