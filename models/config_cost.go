// ConfigCost is an append-only FOCUS-shaped cost line item attached to a config item.
// ConfigCostRollup is the read-only trailing-window aggregate the catalog views join.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/flanksource/duty/types"
)

// Cost bucket grains. Charge periods are snapped to one of these, never split.
const (
	ConfigCostGrainDay   = "day"
	ConfigCostGrainWeek  = "week"
	ConfigCostGrainMonth = "month"
)

// ConfigCost is one cost line item, already merged to its bucket grain, shaped on
// FOCUS v1.4. The period is half-open: [PeriodStart, PeriodEnd), always UTC.
//
// ConfigID is null when the scraped ResourceId could not be resolved to a config item.
// Such rows keep their ExternalID so the compaction job can attach them once the item
// appears, rather than the spend being folded into an account total or dropped.
type ConfigCost struct {
	types.NoOpResourceSelectable `json:"-"`

	ID                      uuid.UUID     `json:"id" gorm:"default:generate_ulid()"`
	ConfigID                *uuid.UUID    `json:"config_id,omitempty"`
	ScraperID               *uuid.UUID    `json:"scraper_id,omitempty"`
	SourceKey               string        `json:"source_key"`
	SourceRecordID          *string       `json:"source_record_id,omitempty"`
	ExternalID              *string       `json:"external_id,omitempty"`
	ExternalConfigType      *string       `json:"external_config_type,omitempty"`
	ExternalConfigScraperID *string       `json:"external_config_scraper_id,omitempty"`
	ExternalConfigLabels    types.JSONMap `json:"external_config_labels,omitempty" gorm:"type:jsonb"`

	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Grain       string    `json:"grain"`

	ChargeCategory   string  `json:"charge_category"`
	ChargeClass      *string `json:"charge_class,omitempty"`
	ServiceName      *string `json:"service_name,omitempty"`
	ServiceCategory  *string `json:"service_category,omitempty"`
	SkuID            *string `json:"sku_id,omitempty"`
	RegionID         *string `json:"region_id,omitempty"`
	BillingAccountID *string `json:"billing_account_id,omitempty"`
	SubAccountID     *string `json:"sub_account_id,omitempty"`
	BillingCurrency  string  `json:"billing_currency"`

	BilledCost      decimal.Decimal  `json:"billed_cost"`
	EffectiveCost   decimal.Decimal  `json:"effective_cost"`
	ListCost        *decimal.Decimal `json:"list_cost,omitempty"`
	ContractedCost  *decimal.Decimal `json:"contracted_cost,omitempty"`
	PricingQuantity *decimal.Decimal `json:"pricing_quantity,omitempty"`
	PricingUnit     *string          `json:"pricing_unit,omitempty"`

	// Focus holds the FOCUS long tail that has no dedicated column: Tags,
	// SkuPriceDetails, CommitmentDiscount*, ContractApplied, Allocated*, and every x_*
	// custom column. FOCUS v1.4 CustomColumnHandling requires x_* columns to survive.
	Focus types.JSONMap `json:"focus,omitempty" gorm:"type:jsonb"`

	// Fingerprint is a deterministic hash of the dimension tuple. Together with
	// (config_id, period_start, period_end) it is the merge key within a bucket.
	Fingerprint string `json:"fingerprint"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ConfigCost) TableName() string {
	return "config_costs"
}

func (c ConfigCost) PK() string {
	return c.ID.String()
}

var _ types.ResourceSelectable = (*ConfigCost)(nil)

func (c ConfigCost) GetFieldsMatcher() fields.Fields {
	m := map[string]any{
		"id":               c.ID.String(),
		"source_key":       c.SourceKey,
		"grain":            c.Grain,
		"charge_category":  c.ChargeCategory,
		"billing_currency": c.BillingCurrency,
		"fingerprint":      c.Fingerprint,
		"billed_cost":      c.BilledCost.String(),
		"effective_cost":   c.EffectiveCost.String(),
		"period_start":     c.PeriodStart,
		"period_end":       c.PeriodEnd,
	}
	if c.ConfigID != nil {
		m["config_id"] = c.ConfigID.String()
	}
	if c.ScraperID != nil {
		m["scraper_id"] = c.ScraperID.String()
	}
	if c.SourceRecordID != nil {
		m["source_record_id"] = *c.SourceRecordID
	}
	if c.ExternalID != nil {
		m["external_id"] = *c.ExternalID
	}
	if c.ExternalConfigType != nil {
		m["external_config_type"] = *c.ExternalConfigType
	}
	if c.ExternalConfigScraperID != nil {
		m["external_config_scraper_id"] = *c.ExternalConfigScraperID
	}
	if c.ServiceName != nil {
		m["service_name"] = *c.ServiceName
	}
	if c.ServiceCategory != nil {
		m["service_category"] = *c.ServiceCategory
	}
	if c.SkuID != nil {
		m["sku_id"] = *c.SkuID
	}
	if c.RegionID != nil {
		m["region_id"] = *c.RegionID
	}
	return types.GenericFieldMatcher{Fields: m}
}

func (c ConfigCost) GetID() string {
	return c.ID.String()
}

func (c ConfigCost) GetName() string {
	if c.ServiceName != nil {
		return *c.ServiceName
	}
	return c.Fingerprint
}

func (c ConfigCost) GetType() string {
	return c.ChargeCategory
}

// ConfigCostRollup is the config_costs_rollup materialized view: trailing-window
// totals per config item and currency, refreshed by refresh_config_costs_rollup(). Read only.
// Column names are spelled out: gorm's naming strategy renders Cost30d as "cost30d",
// which silently binds nothing.
type ConfigCostRollup struct {
	ConfigID        uuid.UUID       `json:"config_id" gorm:"column:config_id"`
	BillingCurrency string          `json:"billing_currency" gorm:"column:billing_currency"`
	Cost1d          decimal.Decimal `json:"cost_1d" gorm:"column:cost_1d"`
	Cost7d          decimal.Decimal `json:"cost_7d" gorm:"column:cost_7d"`
	Cost30d         decimal.Decimal `json:"cost_30d" gorm:"column:cost_30d"`
	Billed30d       decimal.Decimal `json:"billed_30d" gorm:"column:billed_30d"`
	LastCostAt      time.Time       `json:"last_cost_at" gorm:"column:last_cost_at"`
}

func (ConfigCostRollup) TableName() string {
	return "config_costs_rollup"
}
