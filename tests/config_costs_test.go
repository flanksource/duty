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

	AfterAll(func() {
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
		Expect(rollup.BillingCurrency).To(Equal("USD"))
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

	It("keeps currencies in independent rollup rows and nulls legacy mixed totals", func() {
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

		var rollups []models.ConfigCostRollup
		Expect(DefaultContext.DB().Table("config_costs_rollup").
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
				ConfigID:        &configID,
				PeriodStart:     midnight.Add(-24 * time.Hour),
				PeriodEnd:       midnight,
				Grain:           models.ConfigCostGrainDay,
				ChargeCategory:  "Usage",
				BillingCurrency: "USD",
				BilledCost:      decimal.NewFromInt(5),
				EffectiveCost:   decimal.NewFromInt(5),
				Fingerprint:     "shared-fingerprint",
			})
		}

		var count int64
		Expect(DefaultContext.DB().Model(&models.ConfigCost{}).
			Where("config_id = ?", configID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(2)))
		refreshRollup()
		Expect(mustFloat(getRollup(configID).Cost30d)).To(BeNumerically("~", 10, 0.0001))
	})

	It("retains source record identity while merging by its fingerprint", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		sourceRecordID := "invoice-line-17"
		row := models.ConfigCost{
			SourceKey:       "cur-export",
			SourceRecordID:  &sourceRecordID,
			ConfigID:        &configID,
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostGrainDay,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(3),
			EffectiveCost:   decimal.NewFromInt(3),
			Fingerprint:     "source-record/invoice-line-17",
		}
		insertCost(row)
		row.ID = uuid.Nil
		row.BilledCost = decimal.NewFromInt(4)
		row.EffectiveCost = decimal.NewFromInt(4)
		upsertCost(row)

		var stored models.ConfigCost
		Expect(DefaultContext.DB().Where("config_id = ?", configID).First(&stored).Error).To(Succeed())
		Expect(stored.SourceRecordID).ToNot(BeNil())
		Expect(*stored.SourceRecordID).To(Equal(sourceRecordID))
		Expect(mustFloat(stored.EffectiveCost)).To(BeNumerically("~", 4, 0.0001))
	})

	It("enforces source record identity across targets and periods", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		sourceRecordID := "invoice-line-unique"
		row := models.ConfigCost{
			SourceKey:       "cur-export-unique",
			SourceRecordID:  &sourceRecordID,
			ConfigID:        &configID,
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostGrainDay,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(3),
			EffectiveCost:   decimal.NewFromInt(3),
			Fingerprint:     "source-record/invoice-line-unique",
		}
		Expect(DefaultContext.DB().Create(&row).Error).To(Succeed())

		row.ID = uuid.Nil
		row.PeriodStart = row.PeriodStart.Add(-24 * time.Hour)
		row.PeriodEnd = row.PeriodEnd.Add(-24 * time.Hour)
		Expect(DefaultContext.DB().Create(&row).Error).To(MatchError(ContainSubstring("config_costs_source_record_uniq")))
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

	It("rejects structured unresolved identity without an external id", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		externalType := "Kubernetes::Pod"
		cost := models.ConfigCost{
			ID:                 uuid.New(),
			SourceKey:          "kubernetes-cost-export",
			ExternalConfigType: &externalType,
			ExternalConfigLabels: map[string]any{
				"namespace": "payments",
				"name":      "api-0",
			},
			PeriodStart:     midnight.Add(-24 * time.Hour),
			PeriodEnd:       midnight,
			Grain:           models.ConfigCostGrainDay,
			ChargeCategory:  "Usage",
			BillingCurrency: "USD",
			BilledCost:      decimal.NewFromInt(2),
			EffectiveCost:   decimal.NewFromInt(2),
			Fingerprint:     "structured-only",
		}
		Expect(DefaultContext.DB().Create(&cost).Error).To(HaveOccurred())
	})

	It("excludes unmatched spend from the rollup and surfaces it separately", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		externalID := "i-0deadbeef"
		externalType := "AWS::EC2::Instance"
		externalScraperID := "aws-prod"
		sourceRecordID := "line-item-42"
		insertCost(models.ConfigCost{
			SourceKey:               "cur-export-prod",
			SourceRecordID:          &sourceRecordID,
			ExternalID:              &externalID,
			ExternalConfigType:      &externalType,
			ExternalConfigScraperID: &externalScraperID,
			ExternalConfigLabels:    map[string]any{"account": "prod"},
			PeriodStart:             midnight.Add(-24 * time.Hour),
			PeriodEnd:               midnight,
			Grain:                   models.ConfigCostGrainDay,
			ChargeCategory:          "Usage",
			BillingCurrency:         "USD",
			BilledCost:              decimal.NewFromInt(42),
			EffectiveCost:           decimal.NewFromInt(42),
			Fingerprint:             "unmatched",
		})
		refreshRollup()

		var unmatched struct {
			SourceKey               string
			SourceRecordID          string
			ExternalID              string
			ExternalConfigType      string
			ExternalConfigScraperID string
			ExternalConfigLabels    string
			EffectiveCost           decimal.Decimal
		}
		Expect(DefaultContext.DB().Table("config_costs_unmatched").
			Select("source_key, source_record_id, external_id, external_config_type, external_config_scraper_id, external_config_labels::text AS external_config_labels, effective_cost").
			Where("external_id = ?", externalID).Scan(&unmatched).Error).To(Succeed())
		Expect(unmatched.SourceKey).To(Equal("cur-export-prod"))
		Expect(unmatched.SourceRecordID).To(Equal(sourceRecordID))
		Expect(unmatched.ExternalConfigType).To(Equal(externalType))
		Expect(unmatched.ExternalConfigScraperID).To(Equal(externalScraperID))
		Expect(unmatched.ExternalConfigLabels).To(MatchJSON(`{"account":"prod"}`))
		Expect(mustFloat(unmatched.EffectiveCost)).To(BeNumerically("~", 42, 0.0001))

		Expect(DefaultContext.DB().Where("external_id = ?", externalID).
			Delete(&models.ConfigCost{}).Error).To(Succeed())
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

	It("rejects unsupported grains", func() {
		midnight := time.Now().UTC().Truncate(24 * time.Hour)
		cost := models.ConfigCost{
			ConfigID:        &configID,
			PeriodStart:     midnight.Add(-time.Hour),
			PeriodEnd:       midnight,
			Grain:           "hour",
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

func insertCost(c models.ConfigCost) {
	GinkgoHelper()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.SourceKey == "" {
		c.SourceKey = "test"
	}
	Expect(DefaultContext.DB().Create(&c).Error).To(Succeed())
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
		INSERT INTO config_costs (id, source_key, config_id, external_id, period_start, period_end, grain,
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
