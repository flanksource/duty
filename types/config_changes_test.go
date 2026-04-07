package types

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigChangeDetails", func() {
	It("should inject kind into DeploymentDetails", func() {
		d := DeploymentDetails{PreviousImage: "v1.2.3", NewImage: "v1.2.4", Container: "app"}
		data, err := json.Marshal(d)
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("Deployment/v1"))
		Expect(m["previous_image"]).To(Equal("v1.2.3"))
		Expect(m["new_image"]).To(Equal("v1.2.4"))
		Expect(m["container"]).To(Equal("app"))
	})

	It("should inject kind into BackupDetails", func() {
		d := BackupDetails{Status: "success", Size: "4.2GB"}
		data, err := json.Marshal(d)
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("Backup/v1"))
		Expect(m["status"]).To(Equal("success"))
		Expect(m["size"]).To(Equal("4.2GB"))
	})

	It("should inject kind into empty struct", func() {
		d := RollbackDetails{}
		data, err := json.Marshal(d)
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("Rollback/v1"))
		Expect(m).To(HaveLen(1))
	})

	It("should inject kind into ScalingDetails with numeric fields", func() {
		d := ScalingDetails{FromReplicas: 2, ToReplicas: 5, ResourceType: "Deployment"}
		data, err := json.Marshal(d)
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("Scaling/v1"))
		Expect(m["from_replicas"]).To(BeNumerically("==", 2))
		Expect(m["to_replicas"]).To(BeNumerically("==", 5))
	})

	It("should inject kind into CostChangeDetails with float fields", func() {
		d := CostChangeDetails{PreviousCost: 100.50, NewCost: 125.75, Currency: "USD"}
		data, err := json.Marshal(d)
		Expect(err).ToNot(HaveOccurred())

		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		Expect(m["kind"]).To(Equal("CostChange/v1"))
		Expect(m["previous_cost"]).To(BeNumerically("~", 100.50, 0.01))
		Expect(m["new_cost"]).To(BeNumerically("~", 125.75, 0.01))
	})

	DescribeTable("all detail types implement ConfigChangeDetail",
		func(d ConfigChangeDetail, expectedKind string) {
			Expect(d.Kind()).To(Equal(expectedKind))

			data, err := json.Marshal(d)
			Expect(err).ToNot(HaveOccurred())

			var m map[string]any
			Expect(json.Unmarshal(data, &m)).To(Succeed())
			Expect(m["kind"]).To(Equal(expectedKind))
		},
		Entry("UserChange", UserChangeDetails{UserName: "alice"}, "UserChange/v1"),
		Entry("Screenshot", ScreenshotDetails{URL: "https://example.com"}, "Screenshot/v1"),
		Entry("PermissionChange", PermissionChangeDetails{RoleName: "admin"}, "PermissionChange/v1"),
		Entry("Deployment", DeploymentDetails{NewImage: "v2"}, "Deployment/v1"),
		Entry("Promotion", PromotionDetails{ToEnvironment: "prod"}, "Promotion/v1"),
		Entry("Approval", ApprovalDetails{ApprovedBy: "alice"}, "Approval/v1"),
		Entry("Rollback", RollbackDetails{ToVersion: "v1"}, "Rollback/v1"),
		Entry("Backup", BackupDetails{Status: "success"}, "Backup/v1"),
		Entry("PlaybookExecution", PlaybookExecutionDetails{PlaybookName: "restart"}, "PlaybookExecution/v1"),
		Entry("Scaling", ScalingDetails{ToReplicas: 3}, "Scaling/v1"),
		Entry("Certificate", CertificateDetails{Subject: "*.example.com"}, "Certificate/v1"),
		Entry("CostChange", CostChangeDetails{NewCost: 50}, "CostChange/v1"),
		Entry("PipelineRun", PipelineRunDetails{Branch: "main"}, "PipelineRun/v1"),
	)
})
