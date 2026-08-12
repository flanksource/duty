// Verifies config_costs_rollup: exact window sums for day-grain rows and time-weighted
// proration for rows coarser than the window they are being reported into.
package tests

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/shopspring/decimal"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = Describe("config_costs rollup", Ordered, func() {
	var configID uuid.UUID

	// A config item of our own so the fixture rollups stay untouched.
	BeforeAll(func() {
		configID = dummy.EKSCluster.ID
		Expect(DefaultContext.DB().Where("config_id = ?", configID).
			Delete(&models.ConfigCost{}).Error).To(Succeed())
	})

	AfterEach(func() {
		Expect(DefaultContext.DB().Where("config_id = ?", configID).
			Delete(&models.ConfigCost{}).Error).To(Succeed())
		refreshRollup()
	})

	It("sums day-grain rows inside the window exactly", func() {
		// Three consecutive whole days ending at the most recent midnight. All three sit
		// inside the 30d window and outside the 1d window.
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		for i := 1; i <= 3; i++ {
			end := midnight.Add(time.Duration(-(i - 1)) * 24 * time.Hour)
			insertCost(models.ConfigCost{
				ConfigID:        &configID,
				PeriodStart:     end.Add(-24 * time.Hour),
				PeriodEnd:       end,
				Grain:           models.ConfigCostGrainDay,
				ChargeCategory:  "Usage",
				BillingCurrency: "USD",
				BilledCost:      decimal.NewFromInt(10),
				EffectiveCost:   decimal.NewFromInt(10),
				Fingerprint:     "day-sum",
			})
		}
		refreshRollup()

		rollup := getRollup(configID)
		// Compared numerically: numeric(24,10) scaled by the overlap fraction carries a
		// long tail of zeros that says nothing about correctness.
		Expect(mustFloat(rollup.Cost30d)).To(BeNumerically("~", 30, 0.0001))
		Expect(rollup.MixedCurrency).To(BeFalse())
	})

	It("prorates a month-grain row across the shorter windows", func() {
		// $300 over a 30-day month => $10/day. The window boundaries are relative to now(),
		// so the expectation is derived from the same arithmetic the rollup should perform,
		// not read back out of it.
		periodEnd := time.Now().UTC().Truncate(24 * time.Hour)
		periodStart := periodEnd.Add(-30 * 24 * time.Hour)
		insertCost(models.ConfigCost{
			ConfigID:        &configID,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			Grain:           models.ConfigCostGrainMonth,
			ChargeCategory:  "Purchase",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(300),
			EffectiveCost:   decimal.NewFromInt(300),
			Fingerprint:     "month-prorate",
		})
		refreshRollup()

		rollup := getRollup(configID)

		// The period closed at the last midnight but every window is anchored on now(), so
		// each window's tail hangs past the end of the period and contributes nothing.
		// Every window therefore covers its own length minus the time since midnight.
		now := time.Now().UTC()
		elapsed := now.Sub(periodEnd).Hours() // hours since the period closed
		perHour := 300.0 / (30 * 24)

		Expect(mustFloat(rollup.Cost1d)).To(BeNumerically("~", perHour*(24-elapsed), 0.5))
		Expect(mustFloat(rollup.Cost7d)).To(BeNumerically("~", perHour*(7*24-elapsed), 0.5))
		Expect(mustFloat(rollup.Cost30d)).To(BeNumerically("~", perHour*(30*24-elapsed), 0.5))
	})

	It("keeps currencies separate and flags the mix", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		for _, currency := range []string{"USD", "EUR"} {
			insertCost(models.ConfigCost{
				ConfigID:        &configID,
				PeriodStart:     midnight.Add(-24 * time.Hour),
				PeriodEnd:       midnight,
				Grain:           models.ConfigCostGrainDay,
				ChargeCategory:  "Usage",
				BillingCurrency: currency,
				BilledCost:      decimal.NewFromInt(7),
				EffectiveCost:   decimal.NewFromInt(7),
				Fingerprint:     "currency-" + currency,
			})
		}
		refreshRollup()

		Expect(getRollup(configID).MixedCurrency).To(BeTrue())
	})

	It("merges a re-scraped bucket instead of accumulating it", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		row := models.ConfigCost{
			ConfigID:        &configID,
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostGrainDay,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(12),
			EffectiveCost:   decimal.NewFromInt(12),
			Fingerprint:     "idempotent",
		}
		insertCost(row)

		// A provider restating an open billing period: same bucket, same dimensions, a
		// higher running total. It must replace, not add.
		row.ID = uuid.Nil
		row.EffectiveCost = decimal.NewFromInt(19)
		row.BilledCost = decimal.NewFromInt(19)
		upsertCost(row)

		refreshRollup()
		Expect(mustFloat(getRollup(configID).Cost30d)).To(BeNumerically("~", 19, 0.0001))

		var count int64
		Expect(DefaultContext.DB().Model(&models.ConfigCost{}).
			Where("config_id = ?", configID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(1)))
	})

	It("excludes unmatched spend from the rollup and surfaces it separately", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		externalID := "i-0deadbeef"
		insertCost(models.ConfigCost{
			ExternalID:      &externalID,
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostGrainDay,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(42),
			EffectiveCost:   decimal.NewFromInt(42),
			Fingerprint:     "unmatched",
		})
		refreshRollup()

		var unmatched struct {
			ExternalID    string
			EffectiveCost decimal.Decimal
		}
		Expect(DefaultContext.DB().Table("config_costs_unmatched").
			Where("external_id = ?", externalID).Scan(&unmatched).Error).To(Succeed())
		Expect(mustFloat(unmatched.EffectiveCost)).To(BeNumerically("~", 42, 0.0001))

		Expect(DefaultContext.DB().Where("external_id = ?", externalID).
			Delete(&models.ConfigCost{}).Error).To(Succeed())
	})

	It("serves the rollup through the configs view", func() {
		var costTotal30d *float64
		Expect(DefaultContext.DB().Table("configs").
			Select("cost_total_30d").
			Where("id = ?", dummy.KubernetesNodeA.ID).
			Scan(&costTotal30d).Error).To(Succeed())
		Expect(costTotal30d).ToNot(BeNil())
		Expect(*costTotal30d).To(BeNumerically("~", dummy.KubernetesNodeACost30d, 0.0001))
	})
})

func insertCost(c models.ConfigCost) {
	GinkgoHelper()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	Expect(DefaultContext.DB().Create(&c).Error).To(Succeed())
}

// upsertCost applies the same merge the scraper pipeline uses.
func upsertCost(c models.ConfigCost) {
	GinkgoHelper()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	err := DefaultContext.DB().Exec(`
		INSERT INTO config_costs (id, config_id, external_id, period_start, period_end, grain,
		                          charge_category, billing_currency, billed_cost, effective_cost, fingerprint)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (config_id, period_start, period_end, fingerprint)
		DO UPDATE SET billed_cost = excluded.billed_cost,
		              effective_cost = excluded.effective_cost,
		              updated_at = now()`,
		c.ID, c.ConfigID, c.ExternalID, c.PeriodStart, c.PeriodEnd, c.Grain,
		c.ChargeCategory, c.BillingCurrency, c.BilledCost, c.EffectiveCost, c.Fingerprint).Error
	Expect(err).To(Succeed())
}

func refreshRollup() {
	GinkgoHelper()
	Expect(DefaultContext.DB().Exec("SELECT refresh_config_costs_rollup()").Error).To(Succeed())
}

func getRollup(configID uuid.UUID) models.ConfigCostRollup {
	GinkgoHelper()
	var rollup models.ConfigCostRollup
	Expect(DefaultContext.DB().Table("config_costs_rollup").
		Where("config_id = ?", configID).First(&rollup).Error).To(Succeed())
	return rollup
}

func mustFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}
