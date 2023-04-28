package duty

import (
	"context"
	"encoding/json"

	"github.com/flanksource/duty/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		rows, err := testDBPGPool.Query(context.Background(), "SELECT config_class, analysis, changes, total_configs, cost_per_minute, cost_total_1d, cost_total_7d, cost_total_30d FROM config_class_summary")
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

		Expect(configClassSummaries).To(HaveLen(5))
		Expect(configClassSummaries).To(Equal([]configClassSummary{
			{
				configClass:  models.ConfigClassCluster,
				totalConfigs: 2,
				changes:      ptr(2),
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
				totalConfigs: 2,
				changes:      ptr(1),
				cp30d:        ptr(2.5),
			},
			{
				configClass:  models.ConfigClassVirtualMachine,
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
