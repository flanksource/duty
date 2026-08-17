// ConfigCost is a FOCUS-shaped cost line item attached to a config item, and
// ConfigCostCompact is the same row after it has aged out of the hot table.
// ConfigCostSummary is the read-only trailing-window aggregate the catalog views join.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"k8s.io/apimachinery/pkg/fields"

	"github.com/flanksource/duty/types"
)

// Compaction levels. Charge periods are snapped to one of these, never split.
//
// The stored value names the level rather than its width, so the width is an operator
// setting rather than a schema decision — config.costs.level1/2/3, defaulting to 1h, 1d and
// 30d. Changing a width is a property change; it does not rename a column or need a
// migration.
//
// Each level's width must divide the next exactly, so compaction from one to the next is
// pure summation and never splits a row across a boundary. The job validates that before
// it compacts anything.
const (
	ConfigCostLevel1 = "level1"
	ConfigCostLevel2 = "level2"
	ConfigCostLevel3 = "level3"
)

// ConfigCostLevels lists the levels from finest to coarsest.
var ConfigCostLevels = []string{ConfigCostLevel1, ConfigCostLevel2, ConfigCostLevel3}

// ConfigCost is one cost line item, already merged to its bucket grain, shaped on
// FOCUS v1.4. The period is half-open: [PeriodStart, PeriodEnd), always UTC.
//
// ConfigID is always set. Spend with no resource of its own — tax, support, credits, or a
// resource the scraper could not resolve — is attributed by the emitting scraper to its
// root config item (AWS::::Account, GCP::Project, ...). ExternalID is kept as provenance so
// the original resource identifier is still visible on the row.
type ConfigCost struct {
	types.NoOpResourceSelectable `json:"-"`

	ID                      uuid.UUID     `json:"id" gorm:"default:generate_ulid()"`
	ConfigID                uuid.UUID     `json:"config_id"`
	ScraperID               *uuid.UUID    `json:"scraper_id,omitempty"`
	SourceKey               string        `json:"source_key"`
	ExternalID              *string       `json:"external_id,omitempty"`
	ExternalConfigType      *string       `json:"external_config_type,omitempty"`
	ExternalConfigScraperID *string       `json:"external_config_scraper_id,omitempty"`
	ExternalConfigLabels    types.JSONMap `json:"external_config_labels,omitempty" gorm:"type:jsonb"`

	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Grain       string    `json:"grain"`

	ChargeCategory  string  `json:"charge_category"`
	ChargeClass     *string `json:"charge_class,omitempty"`
	ServiceName     *string `json:"service_name,omitempty"`
	ServiceCategory *string `json:"service_category,omitempty"`
	SkuID           *string `json:"sku_id,omitempty"`
	RegionID        *string `json:"region_id,omitempty"`
	BillingCurrency string  `json:"billing_currency"`

	BilledCost      decimal.Decimal  `json:"billed_cost"`
	EffectiveCost   decimal.Decimal  `json:"effective_cost"`
	ListCost        *decimal.Decimal `json:"list_cost,omitempty"`
	ContractedCost  *decimal.Decimal `json:"contracted_cost,omitempty"`
	PricingQuantity *decimal.Decimal `json:"pricing_quantity,omitempty"`
	PricingUnit     *string          `json:"pricing_unit,omitempty"`

	// Focus holds every dimension without a dedicated column: the FOCUS long tail
	// (Tags, SkuPriceDetails, CommitmentDiscount*, ...), all x_* custom columns, and the
	// demoted billing_account_id / sub_account_id.
	//
	// Demoted here does not mean demoted out of identity — the fingerprint still reads
	// the account keys back out by name, so spend from two sub-accounts never merges.
	Focus types.JSONMap `json:"focus,omitempty" gorm:"type:jsonb"`

	// Fingerprint is a deterministic hash of the dimension tuple. Together with
	// (source_key, config_id, period_start, period_end) it is the merge key.
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
		"config_id":        c.ConfigID.String(),
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
	if c.ScraperID != nil {
		m["scraper_id"] = c.ScraperID.String()
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

// ConfigCostCompact is the queryable cost series derived from config_costs, carrying every
// grain. Identical shape, so compaction is a plain INSERT ... SELECT.
type ConfigCostCompact struct {
	ConfigCost
}

func (ConfigCostCompact) TableName() string {
	return "config_cost_compact"
}

// ConfigCostSummary is the config_cost_summary materialized view: trailing-window totals
// per config item and currency, refreshed by refresh_config_cost_summary(). Read only.
//
// Column names are spelled out because gorm's naming strategy renders Cost30d as
// "cost30d", which silently binds nothing.
type ConfigCostSummary struct {
	ConfigID        uuid.UUID       `json:"config_id" gorm:"column:config_id"`
	BillingCurrency string          `json:"billing_currency" gorm:"column:billing_currency"`
	Cost1h          decimal.Decimal `json:"cost_1h" gorm:"column:cost_1h"`
	Cost1d          decimal.Decimal `json:"cost_1d" gorm:"column:cost_1d"`
	Cost30d         decimal.Decimal `json:"cost_30d" gorm:"column:cost_30d"`
	Billed30d       decimal.Decimal `json:"billed_30d" gorm:"column:billed_30d"`
	LastCostAt      time.Time       `json:"last_cost_at" gorm:"column:last_cost_at"`
}

func (ConfigCostSummary) TableName() string {
	return "config_cost_summary"
}
