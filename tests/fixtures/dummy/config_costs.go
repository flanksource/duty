// Cost fixtures for config items. Cost lives in config_costs and reaches the catalog
// through the config_cost_summary materialized view over config_cost_compact, so fixtures
// seed the compacted series rather
// than setting totals on the config item directly.
package dummy

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/flanksource/duty/models"
)

// DayCost builds a single day-grain cost row covering the whole of yesterday (UTC).
// Yesterday sits entirely inside the 30d window, so cost_total_30d comes out to exactly
// `amount` regardless of when the test runs.
func DayCost(configID uuid.UUID, amount string, serviceName string) models.ConfigCostCompact {
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.Add(-24 * time.Hour)
	cost := decimal.RequireFromString(amount)

	return models.ConfigCostCompact{ConfigCost: models.ConfigCost{
		ID:              uuid.New(),
		ConfigID:        configID,
		SourceKey:       "dummy",
		PeriodStart:     start,
		PeriodEnd:       end,
		Grain:           models.ConfigCostLevel2,
		ChargeCategory:  "Usage",
		ServiceName:     &serviceName,
		BillingCurrency: "USD",
		BilledCost:      cost,
		EffectiveCost:   cost,
		Fingerprint:     configID.String() + "/" + serviceName,
	}}
}

// 30-day cost totals the static fixtures roll up to. Exported so assertions can be
// written against the fixture rather than a repeated literal.
const (
	KubernetesNodeACost30d        = 50.0
	KubernetesNodeBCost30d        = 80.0
	KubernetesNodeAKSPool1Cost30d = 100.0
	LogisticsAPIPodCost30d        = 5.0
)

var AllDummyConfigCosts = []models.ConfigCostCompact{
	DayCost(KubernetesNodeA.ID, "50", "Compute"),
	DayCost(KubernetesNodeB.ID, "80", "Compute"),
	DayCost(KubernetesNodeAKSPool1.ID, "100", "Compute"),
	DayCost(LogisticsAPIPodConfig.ID, "5", "Compute"),
}
