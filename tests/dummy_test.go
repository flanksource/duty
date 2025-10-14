package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = ginkgo.Describe("Dummy fixtures with agent annotation", ginkgo.Ordered, func() {
	ginkgo.It("should load echo-server configs with agent set", func() {
		// This test verifies that the ImportConfigs function in @tests/fixtures/dummy/config.go
		// correctly parses the "dummy.flanksource.com/agent" and "dummy.flanksource.com/scraper-id"
		// annotations from Kubernetes resources and sets the AgentID and ScraperID fields on the
		// resulting ConfigItem.
		//
		// The echo-server.yaml fixture contains 3 resources (Deployment, ReplicaSet, Pod) that each
		// have the annotations:
		// - "dummy.flanksource.com/agent: ac4b1dc5-b249-471d-89d7-ba0c5de4997b" (HomelabAgent)
		// - "dummy.flanksource.com/scraper-id: 7f9a2c1d-8b3e-4f5a-9c6d-1e2f3a4b5c6d" (HomelabKubeScraper)
		//
		// We verify that all 3 configs are loaded into the database with the correct AgentID and
		// ScraperID set.

		// UIDs from @tests/fixtures/dummy/config/echo-server.yaml
		deploymentID := uuid.MustParse("c737b53a-4671-48e7-8bbe-92a6432f08f6")
		replicaSetID := uuid.MustParse("2c563bdb-8108-4383-9f51-647094c8c8f7")
		podID := uuid.MustParse("6d7e283b-a918-4734-8eb6-03bc790b3eda")

		// Query database for deployment
		var deployment models.ConfigItem
		err := DefaultContext.DB().Where("id = ?", deploymentID).First(&deployment).Error
		Expect(err).To(BeNil())
		Expect(deployment.ID).To(Equal(deploymentID))
		Expect(deployment.AgentID).To(Equal(dummy.HomelabAgent.ID))
		Expect(*deployment.ScraperID).To(Equal(dummy.HomelabKubeScraper.ID.String()))
		Expect(*deployment.Name).To(Equal("echo-server"))
		Expect(*deployment.Type).To(Equal("Kubernetes::Deployment"))

		// Query database for replicaset
		var replicaSet models.ConfigItem
		err = DefaultContext.DB().Where("id = ?", replicaSetID).First(&replicaSet).Error
		Expect(err).To(BeNil())
		Expect(replicaSet.ID).To(Equal(replicaSetID))
		Expect(replicaSet.AgentID).To(Equal(dummy.HomelabAgent.ID))
		Expect(*replicaSet.ScraperID).To(Equal(dummy.HomelabKubeScraper.ID.String()))
		Expect(*replicaSet.Name).To(Equal("echo-server-685b4476d4"))
		Expect(*replicaSet.Type).To(Equal("Kubernetes::ReplicaSet"))

		// Query database for pod
		var pod models.ConfigItem
		err = DefaultContext.DB().Where("id = ?", podID).First(&pod).Error
		Expect(err).To(BeNil())
		Expect(pod.ID).To(Equal(podID))
		Expect(pod.AgentID).To(Equal(dummy.HomelabAgent.ID))
		Expect(*pod.ScraperID).To(Equal(dummy.HomelabKubeScraper.ID.String()))
		Expect(*pod.Name).To(Equal("echo-server-685b4476d4-9lrc6"))
		Expect(*pod.Type).To(Equal("Kubernetes::Pod"))
	})
})
