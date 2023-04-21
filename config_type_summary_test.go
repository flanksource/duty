package duty

import (
	"context"
	"encoding/json"

	"github.com/flanksource/duty/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type configTypeSummary struct {
	configType            string
	totalConfigs          int
	changes               *int
	cpm, cpd, cp7d, cp30d *float64
	analysis              map[string]any
}

var _ = ginkgo.Describe("Check config_type_summary view", ginkgo.Ordered, func() {
	ginkgo.It("Should query config_type_summary view", func() {
		rows, err := testDBPGPool.Query(context.Background(), "SELECT config_type, analysis, changes, total_configs, cost_per_minute, cost_total_1d, cost_total_7d, cost_total_30d FROM config_type_summary")
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()

		var configTypeSummaries []configTypeSummary
		for rows.Next() {
			var c configTypeSummary
			var analysisRaw json.RawMessage
			err := rows.Scan(&c.configType, &analysisRaw, &c.changes, &c.totalConfigs, &c.cpm, &c.cpd, &c.cp7d, &c.cp30d)
			Expect(err).ToNot(HaveOccurred())

			if analysisRaw != nil {
				err = json.Unmarshal(analysisRaw, &c.analysis)
				Expect(err).ToNot(HaveOccurred())
			}

			configTypeSummaries = append(configTypeSummaries, c)
		}

		Expect(configTypeSummaries).To(HaveLen(5))
		Expect(configTypeSummaries).To(Equal([]configTypeSummary{
			{
				configType:   models.ConfigTypeCluster,
				totalConfigs: 2,
				changes:      ptr(2),
			},
			{
				configType:   models.ConfigTypeDatabase,
				totalConfigs: 1,
				analysis: map[string]any{
					"security": float64(1),
				},
			},
			{
				configType:   models.ConfigTypeDeployment,
				totalConfigs: 3,
			},
			{
				configType:   models.ConfigTypeNode,
				totalConfigs: 2,
				changes:      ptr(1),
				cp30d:        ptr(2.5),
			},
			{
				configType:   models.ConfigTypeVirtualMachine,
				totalConfigs: 2,
				analysis: map[string]any{
					"security": float64(1),
				},
			},
		}),
		)
	})
})

func ptr[T any](i T) *T {
	return &i
}
