package tests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/query"
	"github.com/flanksource/duty/tests/generator"
	"github.com/flanksource/duty/types"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type configClassSummary struct {
	configClass           string
	totalConfigs          int
	changes               *int
	cpm, cpd, cp7d, cp30d *float64
	analysis              map[string]any
}

var _ = ginkgo.Describe("Check config_class_summary view", ginkgo.Ordered, func() {
	ginkgo.It("Should query config_class_summary view", func() {
		rows, err := DefaultContext.Pool().Query(context.Background(), "SELECT config_class, analysis, changes, total_configs, cost_per_minute, cost_total_1d, cost_total_7d, cost_total_30d FROM config_class_summary")
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()

		var configClassSummaries []configClassSummary
		for rows.Next() {
			var c configClassSummary
			var analysisRaw json.RawMessage
			err := rows.Scan(&c.configClass, &analysisRaw, &c.changes, &c.totalConfigs, &c.cpm, &c.cpd, &c.cp7d, &c.cp30d)
			Expect(err).ToNot(HaveOccurred())

			if analysisRaw != nil {
				err = json.Unmarshal(analysisRaw, &c.analysis)
				Expect(err).ToNot(HaveOccurred())
			}

			configClassSummaries = append(configClassSummaries, c)
		}

		expectedSummary := []configClassSummary{
			{
				configClass:  models.ConfigClassCluster,
				totalConfigs: 2,
				changes:      lo.ToPtr(2),
			},
			{
				configClass:  models.ConfigClassDatabase,
				totalConfigs: 1,
				analysis: map[string]any{
					"security": float64(1),
				},
			},
			{
				configClass:  models.ConfigClassDeployment,
				totalConfigs: 3,
			},
			{
				configClass:  models.ConfigClassNode,
				totalConfigs: 3,
				changes:      lo.ToPtr(1),
				cp30d:        lo.ToPtr(2.5),
			},
			{
				configClass:  models.ConfigClassPod,
				totalConfigs: 1,
			},
			{
				configClass:  "ReplicaSet",
				totalConfigs: 1,
			},
			{
				configClass:  models.ConfigClassVirtualMachine,
				totalConfigs: 2,
				analysis: map[string]any{
					"security": float64(1),
				},
			},
		}

		Expect(len(configClassSummaries)).To(BeNumerically(">=", len(expectedSummary)))
		for _, expected := range expectedSummary {

			i, found := lo.Find(configClassSummaries, func(i configClassSummary) bool { return i.configClass == expected.configClass })
			Expect(found).To(BeTrue())
			Expect(i.totalConfigs).To(BeNumerically(">=", expected.totalConfigs))
			Expect(lo.FromPtr(i.changes)).To(BeNumerically(">=", lo.FromPtr(expected.changes)))
		}

	})

	ginkgo.It("Should query config summary by type", func() {
		gen := generator.ConfigGenerator{}
		gen.GenerateConfigItem("Test::type-A", "healthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 4, NumInsightsPerConfig: 3})
		gen.GenerateConfigItem("Test::type-A", "healthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 1, NumInsightsPerConfig: 2})
		gen.GenerateConfigItem("Test::type-A", "unhealthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 1, NumInsightsPerConfig: 2})
		gen.GenerateConfigItem("Test::type-B", "healthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 5, NumInsightsPerConfig: 1})
		gen.GenerateConfigItem("Test::type-B", "healthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 1, NumInsightsPerConfig: 3})
		gen.GenerateConfigItem("Test::type-C", "unhealthy", nil, nil, generator.ConfigTypeRequirements{NumChangesPerConfig: 0, NumInsightsPerConfig: 0})
		for _, item := range gen.Generated.Configs {
			DefaultContext.DB().Create(&item)
		}
		for _, item := range gen.Generated.Changes {
			DefaultContext.DB().Create(&item)
		}
		for i, item := range gen.Generated.Analysis {
			if i%3 == 0 {
				item.AnalysisType = models.AnalysisTypeCost
			}
			if i%3 == 1 {
				item.AnalysisType = models.AnalysisTypeSecurity
			}
			DefaultContext.DB().Create(&item)
		}

		err := job.RefreshConfigItemSummary30d(DefaultContext)
		Expect(err).To(BeNil())

		summary30D, err := query.ConfigSummary(DefaultContext, query.ConfigSummaryRequest{
			GroupBy: []string{"type"},
			Changes: query.ConfigSummaryRequestChanges{Since: "30d"},
		})
		Expect(err).To(BeNil())

		type summaryRow struct {
			Type     string
			Count    int
			Health   types.JSONMap
			Changes  int
			Analysis types.JSONMap
		}
		var rows []summaryRow
		err = json.Unmarshal(summary30D, &rows)
		Expect(err).To(BeNil())

		expectedTypeSummary := []summaryRow{
			{Type: "Test::type-A", Count: 3, Changes: 9, Health: map[string]any{"healthy": float64(2), "unhealthy": float64(1)}, Analysis: map[string]any{"availability": float64(2), "cost": float64(3), "security": float64(2)}},
			{Type: "Test::type-B", Count: 2, Changes: 8, Health: map[string]any{"healthy": float64(2)}, Analysis: map[string]any{"availability": float64(1), "cost": float64(1), "security": float64(2)}},
			{Type: "Test::type-C", Count: 1, Changes: 1, Health: map[string]any{"unhealthy": float64(1)}, Analysis: map[string]any{}},
		}

		for _, expected := range expectedTypeSummary {
			i, found := lo.Find(rows, func(i summaryRow) bool { return i.Type == expected.Type })
			Expect(found).To(BeTrue())
			Expect(i.Count).To(Equal(expected.Count))
			Expect(i.Changes).To(Equal(expected.Changes))
			Expect(i.Health).To(Equal(expected.Health))
			Expect(i.Analysis).To(Equal(expected.Analysis))
		}

		// We are making a change and an analysis older than 7 days, it should reflect in the summary
		change0 := gen.Generated.Changes[0]
		err = DefaultContext.DB().Model(&models.ConfigChange{}).Where("id = ?", change0.ID).UpdateColumns(map[string]any{
			"created_at": gorm.Expr("NOW() - '15 days'::interval"),
			"details":    `{"reason": "test reason"}`,
		}).Error
		Expect(err).To(BeNil())

		analysis0 := gen.Generated.Analysis[0]
		DefaultContext.DB().Model(&models.ConfigAnalysis{}).Where("id = ?", analysis0.ID).UpdateColumn("status", "closed")

		err = job.RefreshConfigItemSummary7d(DefaultContext)
		Expect(err).To(BeNil())
		summary7D, err := query.ConfigSummary(DefaultContext, query.ConfigSummaryRequest{
			GroupBy: []string{"type"},
			Changes: query.ConfigSummaryRequestChanges{Since: "7d"},
		})
		Expect(err).To(BeNil())

		var rows7d []summaryRow
		err = json.Unmarshal(summary7D, &rows7d)
		Expect(err).To(BeNil())

		expectedTypeSummary7d := []summaryRow{
			{Type: "Test::type-A", Count: 3, Changes: 8, Health: map[string]any{"healthy": float64(2), "unhealthy": float64(1)}, Analysis: map[string]any{"availability": float64(2), "cost": float64(2), "security": float64(2)}},
			{Type: "Test::type-B", Count: 2, Changes: 8, Health: map[string]any{"healthy": float64(2)}, Analysis: map[string]any{"availability": float64(1), "cost": float64(1), "security": float64(2)}},
			{Type: "Test::type-C", Count: 1, Changes: 1, Health: map[string]any{"unhealthy": float64(1)}, Analysis: map[string]any{}},
		}

		for _, expected := range expectedTypeSummary7d {
			i, found := lo.Find(rows7d, func(i summaryRow) bool { return i.Type == expected.Type })
			Expect(found).To(BeTrue())
			Expect(i.Count).To(Equal(expected.Count), fmt.Sprintf("count mismatched for type %s", expected.Type))
			Expect(i.Changes).To(Equal(expected.Changes), fmt.Sprintf("changes count mismatched for type %s", expected.Type))
			Expect(i.Health).To(Equal(expected.Health), fmt.Sprintf("health mismatched for type %s", expected.Type))
			Expect(i.Analysis).To(Equal(expected.Analysis), fmt.Sprintf("analysis count mismatched for type %s", expected.Type))
		}

		err = job.RefreshConfigItemSummary3d(DefaultContext)
		Expect(err).To(BeNil())
		summary3D, err := query.ConfigSummary(DefaultContext, query.ConfigSummaryRequest{
			GroupBy: []string{"type"},
			Changes: query.ConfigSummaryRequestChanges{Since: "3d"},
		})
		Expect(err).To(BeNil())

		var rows3d []summaryRow
		err = json.Unmarshal(summary3D, &rows3d)
		Expect(err).To(BeNil())

		expectedTypeSummary3d := []summaryRow{
			{Type: "Test::type-A", Count: 3, Changes: 8, Health: map[string]any{"healthy": float64(2), "unhealthy": float64(1)}, Analysis: map[string]any{"availability": float64(2), "cost": float64(2), "security": float64(2)}},
			{Type: "Test::type-B", Count: 2, Changes: 8, Health: map[string]any{"healthy": float64(2)}, Analysis: map[string]any{"availability": float64(1), "cost": float64(1), "security": float64(2)}},
			{Type: "Test::type-C", Count: 1, Changes: 1, Health: map[string]any{"unhealthy": float64(1)}, Analysis: map[string]any{}},
		}

		for _, expected := range expectedTypeSummary3d {
			i, found := lo.Find(rows3d, func(i summaryRow) bool { return i.Type == expected.Type })
			Expect(found).To(BeTrue())
			Expect(i.Count).To(Equal(expected.Count), fmt.Sprintf("count mismatched for type %s", expected.Type))
			Expect(i.Changes).To(Equal(expected.Changes), fmt.Sprintf("changes count mismatched for type %s", expected.Type))
			Expect(i.Health).To(Equal(expected.Health), fmt.Sprintf("health mismatched for type %s", expected.Type))
			Expect(i.Analysis).To(Equal(expected.Analysis), fmt.Sprintf("analysis count mismatched for type %s", expected.Type))
		}

		// We have separate test for 10 days since 30day, 7day and 3day summary is served from materialized views
		// but any other time range is served from a view
		summary10D, err := query.ConfigSummary(DefaultContext, query.ConfigSummaryRequest{
			GroupBy: []string{"type"},
			Changes: query.ConfigSummaryRequestChanges{Since: "10d"},
		})
		Expect(err).To(BeNil())

		var rows10d []summaryRow
		err = json.Unmarshal(summary10D, &rows10d)
		Expect(err).To(BeNil())

		expectedTypeSummary10d := []summaryRow{
			{Type: "Test::type-A", Count: 3, Changes: 8, Health: map[string]any{"healthy": float64(2), "unhealthy": float64(1)}, Analysis: map[string]any{"availability": float64(2), "cost": float64(2), "security": float64(2)}},
			{Type: "Test::type-B", Count: 2, Changes: 8, Health: map[string]any{"healthy": float64(2)}, Analysis: map[string]any{"availability": float64(1), "cost": float64(1), "security": float64(2)}},
			{Type: "Test::type-C", Count: 1, Changes: 1, Health: map[string]any{"unhealthy": float64(1)}, Analysis: nil},
		}

		for _, expected := range expectedTypeSummary10d {
			i, found := lo.Find(rows10d, func(i summaryRow) bool { return i.Type == expected.Type })
			Expect(found).To(BeTrue())
			Expect(i.Count).To(Equal(expected.Count), fmt.Sprintf("count mismatched for type %s", expected.Type))
			Expect(i.Changes).To(Equal(expected.Changes), fmt.Sprintf("changes count mismatched for type %s", expected.Type))
			Expect(i.Health).To(Equal(expected.Health), fmt.Sprintf("health mismatched for type %s", expected.Type))
			Expect(i.Analysis).To(Equal(expected.Analysis), fmt.Sprintf("analysis count mismatched for type %s", expected.Type))
		}
	})
})
