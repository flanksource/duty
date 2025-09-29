package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/dataquery"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/flanksource/duty/view"
)

var _ = Describe("View Query Tests", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = DefaultContext
	})

	Describe("ExecuteQuery", func() {
		Context("with configs query", func() {
			It("should execute configs query successfully", func() {
				query := view.Query{
					Configs: &types.ResourceSelector{
						Name: "node-a",
					},
				}

				results, err := view.ExecuteQuery(ctx, query)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0]["name"]).To(Equal("node-a"))
			})
		})

		Context("with changes query", func() {
			It("should execute changes query successfully", func() {
				query := view.Query{
					Changes: &types.ResourceSelector{
						Search: "change_type=CREATE",
					},
				}

				results, err := view.ExecuteQuery(ctx, query)
				Expect(err).ToNot(HaveOccurred())
				changeIDs := lo.Map(results, func(result dataquery.QueryResultRow, _ int) string {
					return result["id"].(string)
				})
				Expect(changeIDs).To(ConsistOf(dummy.EKSClusterCreateChange.ID, dummy.KubernetesNodeAChange.ID))
				Expect(results).To(HaveLen(2))
			})
		})

		Context("with empty query", func() {
			It("should handle empty query by falling back to dataquery", func() {
				query := view.Query{}
				_, err := view.ExecuteQuery(ctx, query)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

var _ = Describe("QueryViewTables", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = DefaultContext
	})

	Context("with valid view selector", func() {
		It("should query view tables successfully", func() {
			selector := view.ViewSelector{
				Name: "test-view",
			}

			results, err := view.QueryViewTables(ctx, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})
	})

	Context("with empty view selector", func() {
		It("should handle empty selector", func() {
			selector := view.ViewSelector{}

			results, err := view.QueryViewTables(ctx, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})
	})

	Context("with nonexistent view", func() {
		It("should handle nonexistent view gracefully", func() {
			selector := view.ViewSelector{
				Name: "nonexistent-view",
			}

			results, err := view.QueryViewTables(ctx, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})
	})

	Context("with label selector", func() {
		It("should query views by label selector", func() {
			selector := view.ViewSelector{
				LabelSelector: "environment=test",
			}

			results, err := view.QueryViewTables(ctx, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})
	})

	Context("with namespace selector", func() {
		It("should query views by namespace", func() {
			selector := view.ViewSelector{
				Namespace: "test-namespace",
			}

			results, err := view.QueryViewTables(ctx, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(0))
		})
	})
})
