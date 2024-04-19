package tests

import (
	"encoding/json"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Config Summary Search", ginkgo.Ordered, func() {
	ginkgo.It("should query config summary", func() {
		request := query.ConfigSummaryRequest{}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		types := lo.Map(output, func(item map[string]any, _ int) string {
			return item["type"].(string)
		})
		expected := lo.Uniq(lo.Map(dummy.AllDummyConfigs, func(item models.ConfigItem, _ int) string {
			return lo.FromPtr(item.Type)
		}))
		Expect(types).To(ConsistOf(expected))
	})

	ginkgo.It("should not fetch analysis and changes if not requested", func() {
		request := query.ConfigSummaryRequest{}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		for _, o := range output {
			_, ok := o["analysis"]
			Expect(ok).To(BeFalse())

			_, ok = o["changes"]
			Expect(ok).To(BeFalse())
		}
	})

	ginkgo.It("should filter by labels", func() {
		request := query.ConfigSummaryRequest{
			Filter: map[string]string{
				"environment": "production",
			},
		}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		types := lo.Map(output, func(item map[string]any, _ int) string {
			return item["type"].(string)
		})
		withLabels := lo.Filter(dummy.AllDummyConfigs, func(item models.ConfigItem, _ int) bool {
			return lo.FromPtr(item.Labels)["environment"] == "production"
		})
		expected := lo.Uniq(lo.Map(withLabels, func(item models.ConfigItem, _ int) string {
			return lo.FromPtr(item.Type)
		}))
		Expect(types).To(ConsistOf(expected))
	})

	ginkgo.Context("should query changes by range", func() {
		ginkgo.It("small range", func() {
			request := query.ConfigSummaryRequest{
				Changes: query.ConfigSummaryRequestChanges{
					Since: "7d",
				},
				Filter: lo.FromPtr(dummy.EKSCluster.Labels),
			}
			response, err := query.ConfigSummary(DefaultContext, request)
			Expect(err).To(BeNil())

			var output []map[string]any
			err = json.Unmarshal(response, &output)
			Expect(err).To(BeNil())
			Expect(len(output)).To(Equal(1))
			Expect(output[0]["type"].(string)).To(Equal(lo.FromPtr(dummy.EKSCluster.Type)))
			Expect(output[0]["changes"].(float64)).To(Equal(float64(2)))
		})

		ginkgo.It("large range", func() {
			request := query.ConfigSummaryRequest{
				Changes: query.ConfigSummaryRequestChanges{
					Since: "5y",
				},
				Filter: lo.FromPtr(dummy.EKSCluster.Labels),
			}
			response, err := query.ConfigSummary(DefaultContext, request)
			Expect(err).To(BeNil())

			var output []map[string]any
			err = json.Unmarshal(response, &output)
			Expect(err).To(BeNil())
			Expect(len(output)).To(Equal(1))
			Expect(output[0]["type"].(string)).To(Equal(lo.FromPtr(dummy.EKSCluster.Type)))
			Expect(output[0]["changes"].(float64)).To(Equal(float64(3)))
		})
	})

	ginkgo.It("should return queried tags as columns", func() {
		request := query.ConfigSummaryRequest{
			Tags: []string{"cluster"},
		}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		itemsWithCluster := lo.Filter(output, func(item map[string]any, _ int) bool {
			val, ok := item["cluster"]
			return ok && val != "" && val != nil
		})
		got := lo.Uniq(lo.Map(itemsWithCluster, func(item map[string]any, _ int) string {
			val, _ := item["cluster"].(string)
			return val
		}))

		withLabels := lo.Filter(dummy.AllDummyConfigs, func(item models.ConfigItem, _ int) bool {
			val, ok := lo.FromPtr(item.Labels)["cluster"]
			return ok && val != ""
		})
		expected := lo.Uniq(lo.Map(withLabels, func(item models.ConfigItem, _ int) string {
			val, _ := lo.FromPtr(item.Labels)["cluster"] //nolint: gosimple
			return val
		}))

		Expect(got).To(ConsistOf(expected))
	})

	ginkgo.It("should group by account", func() {
		request := query.ConfigSummaryRequest{GroupBy: []string{"account"}}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		itemsWithAccount := lo.Filter(output, func(item map[string]any, _ int) bool {
			val, ok := item["account"]
			return ok && val != "" && val != nil
		})
		got := lo.Uniq(lo.Map(itemsWithAccount, func(item map[string]any, _ int) string {
			val, _ := item["account"].(string)
			return val
		}))

		withLabels := lo.Filter(dummy.AllDummyConfigs, func(item models.ConfigItem, _ int) bool {
			val, ok := lo.FromPtr(item.Labels)["account"]
			return ok && val != ""
		})
		expected := lo.Uniq(lo.Map(withLabels, func(item models.ConfigItem, _ int) string {
			val, _ := lo.FromPtr(item.Labels)["account"] //nolint: gosimple
			return val
		}))
		Expect(got).To(ConsistOf(expected))
	})

	ginkgo.It("should group by account & type", func() {
		request := query.ConfigSummaryRequest{GroupBy: []string{"account", "type"}}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		accountFlanksource := lo.Filter(output, func(item map[string]any, _ int) bool {
			ac, ok := item["account"]
			return ok && ac != nil && ac == "flanksource"
		})

		types := lo.Uniq(lo.Map(accountFlanksource, func(item map[string]any, _ int) string {
			return item["type"].(string)
		}))

		Expect(types).To(ConsistOf([]string{"EC2::Instance", "EKS::Cluster", "Kubernetes::Node", "Kubernetes::Cluster"}))
	})

	ginkgo.It("should fetch health summary", func() {
		request := query.ConfigSummaryRequest{
			Filter: map[string]string{
				"role": "worker",
			},
		}
		response, err := query.ConfigSummary(DefaultContext, request)
		Expect(err).To(BeNil())

		var output []map[string]any
		err = json.Unmarshal(response, &output)
		Expect(err).To(BeNil())

		Expect(len(output)).To(Equal(1))
		Expect(output[0]["type"].(string)).To(Equal("Kubernetes::Node"))

		summary, ok := output[0]["health_summary"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(summary["Healthy"]).To(Equal(float64(2)))
	})
})
