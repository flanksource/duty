// Verifies config_cost_summary over config_cost_compact: exact window sums and time-weighted
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

var _ = Describe("config cost summary", Ordered, func() {
	var configID uuid.UUID

	// A config item of our own so the fixture rollups stay untouched.
	BeforeAll(func() {
		configID = dummy.EKSCluster.ID
		Expect(DefaultContext.DB().Where("config_id = ?", configID).
			Delete(&models.ConfigCostCompact{}).Error).To(Succeed())
	})

	AfterAll(func() {
		Expect(DefaultContext.DB().Where("config_id = ?", configID).
			Delete(&models.ConfigCostCompact{}).Error).To(Succeed())
	})

	AfterEach(func() {
		Expect(DefaultContext.DB().Where("config_id = ?", configID).
			Delete(&models.ConfigCostCompact{}).Error).To(Succeed())
		refreshSummary()
	})

	It("sums day-grain rows inside the window exactly", func() {
		// Three consecutive whole days ending at the most recent midnight. All three sit
		// inside the 30d window and outside the 1d window.
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		for i := 1; i <= 3; i++ {
			end := midnight.Add(time.Duration(-(i - 1)) * 24 * time.Hour)
			insertCost(models.ConfigCost{
				ConfigID:        configID,
				PeriodStart:     end.Add(-24 * time.Hour),
				PeriodEnd:       end,
				Grain:           models.ConfigCostLevel2,
				ChargeCategory:  "Usage",
				BillingCurrency: "USD",
				BilledCost:      decimal.NewFromInt(10),
				EffectiveCost:   decimal.NewFromInt(10),
				Fingerprint:     "day-sum",
			})
		}
		refreshSummary()

		rollup := getSummary(configID)
		// Compared numerically: numeric(24,10) scaled by the overlap fraction carries a
		// long tail of zeros that says nothing about correctness.
		Expect(mustFloat(rollup.Cost30d)).To(BeNumerically("~", 30, 0.0001))
		Expect(rollup.BillingCurrency).To(Equal("USD"))
	})

	It("prorates a month-grain row across the shorter windows", func() {
		// $300 over a 30-day month => $10/day. The window boundaries are relative to now(),
		// so the expectation is derived from the same arithmetic the rollup should perform,
		// not read back out of it.
		periodEnd := time.Now().UTC().Truncate(24 * time.Hour)
		periodStart := periodEnd.Add(-30 * 24 * time.Hour)
		insertCost(models.ConfigCost{
			ConfigID:        configID,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
			Grain:           models.ConfigCostLevel3,
			ChargeCategory:  "Purchase",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(300),
			EffectiveCost:   decimal.NewFromInt(300),
			Fingerprint:     "month-prorate",
		})
		refreshSummary()

		rollup := getSummary(configID)

		// The period closed at the last midnight but every window is anchored on now(), so
		// each window's tail hangs past the end of the period and contributes nothing.
		// Every window therefore covers its own length minus the time since midnight.
		now := time.Now().UTC()
		elapsed := now.Sub(periodEnd).Hours() // hours since the period closed
		perHour := 300.0 / (30 * 24)

		Expect(mustFloat(rollup.Cost1d)).To(BeNumerically("~", perHour*(24-elapsed), 0.5))
		Expect(mustFloat(rollup.Cost30d)).To(BeNumerically("~", perHour*(30*24-elapsed), 0.5))
	})

	It("keeps currencies in independent rollup rows and nulls legacy mixed totals", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		for _, currency := range []string{"USD", "EUR"} {
			insertCost(models.ConfigCost{
				ConfigID:        configID,
				PeriodStart:     midnight.Add(-24 * time.Hour),
				PeriodEnd:       midnight,
				Grain:           models.ConfigCostLevel2,
				ChargeCategory:  "Usage",
				BillingCurrency: currency,
				BilledCost:      decimal.NewFromInt(7),
				EffectiveCost:   decimal.NewFromInt(7),
				Fingerprint:     "currency-" + currency,
			})
		}
		refreshSummary()

		var rollups []models.ConfigCostSummary
		Expect(DefaultContext.DB().Table("config_cost_summary").
			Where("config_id = ?", configID).Order("billing_currency").Find(&rollups).Error).To(Succeed())
		Expect(rollups).To(HaveLen(2))
		Expect([]string{rollups[0].BillingCurrency, rollups[1].BillingCurrency}).To(Equal([]string{"EUR", "USD"}))
		for _, rollup := range rollups {
			Expect(mustFloat(rollup.Cost30d)).To(BeNumerically("~", 7, 0.0001))
		}

		var config struct {
			CostTotal30d    *float64
			BillingCurrency *string
			MixedCurrency   bool
		}
		Expect(DefaultContext.DB().Table("configs").
			Select("cost_total_30d, billing_currency, mixed_currency").
			Where("id = ?", configID).Scan(&config).Error).To(Succeed())
		Expect(config.CostTotal30d).To(BeNil())
		Expect(config.BillingCurrency).To(BeNil())
		Expect(config.MixedCurrency).To(BeTrue())

		var identity struct {
			Type        string
			ConfigClass string
		}
		Expect(DefaultContext.DB().Table("config_items").Select("type, config_class").
			Where("id = ?", configID).Scan(&identity).Error).To(Succeed())
		var typeTotal, classTotal *float64
		Expect(DefaultContext.DB().Table("config_summary").Select("cost_total_30d").
			Where("type = ?", identity.Type).Scan(&typeTotal).Error).To(Succeed())
		Expect(DefaultContext.DB().Table("config_class_summary").Select("cost_total_30d").
			Where("config_class = ?", identity.ConfigClass).Scan(&classTotal).Error).To(Succeed())
		Expect(typeTotal).To(BeNil())
		Expect(classTotal).To(BeNil())
	})

	It("keeps identical rows from different sources distinct", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		for _, sourceKey := range []string{"cur-export-a", "cur-export-b"} {
			insertCost(models.ConfigCost{
				SourceKey:       sourceKey,
				ConfigID:        configID,
				PeriodStart:     midnight.Add(-24 * time.Hour),
				PeriodEnd:       midnight,
				Grain:           models.ConfigCostLevel2,
				ChargeCategory:  "Usage",
				BillingCurrency: "USD",
				BilledCost:      decimal.NewFromInt(5),
				EffectiveCost:   decimal.NewFromInt(5),
				Fingerprint:     "shared-fingerprint",
			})
		}

		var count int64
		Expect(DefaultContext.DB().Model(&models.ConfigCostCompact{}).
			Where("config_id = ?", configID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(2)))
		refreshSummary()
		Expect(mustFloat(getSummary(configID).Cost30d)).To(BeNumerically("~", 10, 0.0001))
	})

	It("merges a re-scraped bucket instead of accumulating it", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		row := models.ConfigCost{
			ConfigID:        configID,
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostLevel2,
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

		refreshSummary()
		Expect(mustFloat(getSummary(configID).Cost30d)).To(BeNumerically("~", 19, 0.0001))

		var count int64
		Expect(DefaultContext.DB().Model(&models.ConfigCostCompact{}).
			Where("config_id = ?", configID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(1)))
	})

	It("serves a single-currency rollup through one configs row", func() {
		var config models.ConfigItemSummary
		Expect(DefaultContext.DB().Table("configs").
			Where("id = ?", dummy.KubernetesNodeA.ID).
			Scan(&config).Error).To(Succeed())
		Expect(config.CostTotal30d).ToNot(BeNil())
		Expect(*config.CostTotal30d).To(BeNumerically("~", dummy.KubernetesNodeACost30d, 0.0001))
		Expect(config.BillingCurrency).ToNot(BeNil())
		Expect(*config.BillingCurrency).To(Equal("USD"))
		Expect(config.MixedCurrency).To(BeFalse())

		var count int64
		Expect(DefaultContext.DB().Table("configs").Where("id = ?", dummy.KubernetesNodeA.ID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(1)))
	})

	It("serves the same rollup through config_detail", func() {
		// config_detail backs the catalog sidebar. It projects config_items.*, which no
		// longer carries cost, so the totals have to be joined in the way configs joins
		// them — otherwise the sidebar reports no cost at all and says nothing about it.
		var detail models.ConfigItemSummary
		Expect(DefaultContext.DB().Table("config_detail").
			Where("id = ?", dummy.KubernetesNodeA.ID).
			Scan(&detail).Error).To(Succeed())
		Expect(detail.CostTotal30d).ToNot(BeNil())
		Expect(*detail.CostTotal30d).To(BeNumerically("~", dummy.KubernetesNodeACost30d, 0.0001))
		Expect(detail.BillingCurrency).ToNot(BeNil())
		Expect(*detail.BillingCurrency).To(Equal("USD"))
		Expect(detail.MixedCurrency).To(BeFalse())
	})

	It("reports a level-1 row in cost_1h", func() {
		// A level-1 bucket (an hour by default) ending now sits inside every window, so
		// the wider totals are exact.
		hour := time.Now().UTC().Truncate(time.Hour)
		insertCost(models.ConfigCost{
			ConfigID:        configID,
			PeriodStart:     hour.Add(-time.Hour),
			PeriodEnd:       hour,
			Grain:           models.ConfigCostLevel1,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(3),
			EffectiveCost:   decimal.NewFromInt(3),
			Fingerprint:     "hour-grain",
		})
		refreshSummary()

		summary := getSummary(configID)
		elapsed := time.Since(hour).Hours() // the bucket closed this long ago
		Expect(mustFloat(summary.Cost1h)).To(BeNumerically("~", 3*(1-elapsed), 0.1))
		Expect(mustFloat(summary.Cost1d)).To(BeNumerically("~", 3, 0.0001))
		Expect(mustFloat(summary.Cost30d)).To(BeNumerically("~", 3, 0.0001))
	})

	It("accepts every level in the ladder", func() {
		hour := time.Now().UTC().Truncate(time.Hour)
		for i, grain := range models.ConfigCostLevels {
			cost := models.ConfigCost{
				ID:              uuid.New(),
				SourceKey:       "grain-ladder",
				ConfigID:        configID,
				PeriodStart:     hour.Add(time.Duration(-i-1) * time.Hour),
				PeriodEnd:       hour.Add(time.Duration(-i) * time.Hour),
				Grain:           grain,
				ChargeCategory:  "Usage",
				BillingCurrency: "USD",
				BilledCost:      decimal.NewFromInt(1),
				EffectiveCost:   decimal.NewFromInt(1),
				Fingerprint:     "ladder-" + grain,
			}
			Expect(DefaultContext.DB().Create(&cost).Error).To(Succeed(), "level %s should be accepted", grain)
		}
	})

	It("stores aged-out rows in config_cost_compact under the same merge key", func() {
		bucket := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -120)
		row := models.ConfigCostCompact{ConfigCost: models.ConfigCost{
			ID:              uuid.New(),
			SourceKey:       "compact-test",
			ConfigID:        configID,
			PeriodStart:     bucket,
			PeriodEnd:       bucket.AddDate(0, 0, 30),
			Grain:           models.ConfigCostLevel3,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(900),
			EffectiveCost:   decimal.NewFromInt(900),
			Fingerprint:     "compacted",
		}}
		Expect(DefaultContext.DB().Create(&row).Error).To(Succeed())
		DeferCleanup(func() {
			Expect(DefaultContext.DB().Where("source_key = ?", "compact-test").
				Delete(&models.ConfigCostCompact{}).Error).To(Succeed())
		})

		// Re-inserting the same bucket must collide, not duplicate.
		duplicate := row
		duplicate.ID = uuid.New()
		Expect(DefaultContext.DB().Create(&duplicate).Error).
			To(MatchError(ContainSubstring("config_cost_compact_merge_uniq")))

		// Compacted history is deliberately outside every summary window.
		refreshSummary()
		var summary models.ConfigCostSummary
		err := DefaultContext.DB().Table("config_cost_summary").
			Where("config_id = ?", configID).First(&summary).Error
		if err == nil {
			Expect(mustFloat(summary.Cost30d)).To(BeNumerically("<", 900))
		}
	})

	It("rejects a grain outside the level ladder", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		cost := models.ConfigCost{
			ConfigID:        configID,
			PeriodStart:     midnight.Add(-time.Hour),
			PeriodEnd:       midnight,
			Grain:           "week",
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(1),
			EffectiveCost:   decimal.NewFromInt(1),
			Fingerprint:     "invalid-grain",
		}
		cost.ID = uuid.New()
		cost.SourceKey = "test"
		Expect(DefaultContext.DB().Create(&cost).Error).To(HaveOccurred())
	})
})

// insertCost seeds the compacted series, which is what config_cost_summary reads.
// config_costs is the raw landing zone and nothing queries it directly.
func insertCost(c models.ConfigCost) {
	GinkgoHelper()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.SourceKey == "" {
		c.SourceKey = "test"
	}
	Expect(DefaultContext.DB().Create(&models.ConfigCostCompact{ConfigCost: c}).Error).To(Succeed())
}

// upsertCost applies the same merge the scraper pipeline uses.
func upsertCost(c models.ConfigCost) {
	GinkgoHelper()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.SourceKey == "" {
		c.SourceKey = "test"
	}
	err := DefaultContext.DB().Exec(`
		INSERT INTO config_cost_compact (id, source_key, config_id, external_id, period_start, period_end, grain,
		                          charge_category, billing_currency, billed_cost, effective_cost, fingerprint)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (source_key, config_id, period_start, period_end, fingerprint)
		DO UPDATE SET billed_cost = excluded.billed_cost,
		              effective_cost = excluded.effective_cost,
		              updated_at = now()`,
		c.ID, c.SourceKey, c.ConfigID, c.ExternalID, c.PeriodStart, c.PeriodEnd, c.Grain,
		c.ChargeCategory, c.BillingCurrency, c.BilledCost, c.EffectiveCost, c.Fingerprint).Error
	Expect(err).To(Succeed())
}

func refreshSummary() {
	GinkgoHelper()
	Expect(DefaultContext.DB().Exec("SELECT refresh_config_cost_summary()").Error).To(Succeed())
}

func getSummary(configID uuid.UUID) models.ConfigCostSummary {
	GinkgoHelper()
	var summary models.ConfigCostSummary
	Expect(DefaultContext.DB().Table("config_cost_summary").
		Where("config_id = ?", configID).First(&summary).Error).To(Succeed())
	return summary
}

func mustFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}
