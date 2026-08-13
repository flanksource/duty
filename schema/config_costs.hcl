table "config_costs" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "config_id" {
    null    = true
    type    = uuid
    comment = "null when the spend could not be attached to a config item"
  }
  column "scraper_id" {
    null    = true
    type    = uuid
    comment = "config scraper that emitted this cost row"
  }
  column "source_key" {
    null    = false
    type    = text
    comment = "immutable identity of the source dataset or connection"
  }
  column "source_record_id" {
    null    = true
    type    = text
    comment = "immutable source-native record identity, unique within source_key; defines the fingerprint when present"
  }
  column "external_id" {
    null    = true
    type    = text
    comment = "legacy external resource identifier retained for matching"
  }
  column "external_config_type" {
    null = true
    type = text
  }
  column "external_config_scraper_id" {
    null = true
    type = text
  }
  column "external_config_labels" {
    null = true
    type = jsonb
  }

  # half-open [period_start, period_end), clock-aligned, always UTC
  column "period_start" {
    null = false
    type = timestamptz
  }
  column "period_end" {
    null = false
    type = timestamptz
  }
  column "grain" {
    null    = false
    type    = text
    comment = "day | week | month"
  }

  # FOCUS dimensions kept queryable
  column "charge_category" {
    null    = false
    type    = text
    comment = "Usage | Purchase | Tax | Credit | Adjustment"
  }
  column "charge_class" {
    null    = true
    type    = text
    comment = "Correction, or null for an original charge"
  }
  column "service_name" {
    null = true
    type = text
  }
  column "service_category" {
    null = true
    type = text
  }
  column "sku_id" {
    null = true
    type = text
  }
  column "region_id" {
    null = true
    type = text
  }
  column "billing_account_id" {
    null = true
    type = text
  }
  column "sub_account_id" {
    null = true
    type = text
  }
  column "billing_currency" {
    null    = false
    type    = text
    comment = "ISO 4217"
  }

  # FOCUS metrics. NUMERIC rather than float: real billing data carries 6+ decimal places
  # and summing floats over a month accumulates visible error.
  column "billed_cost" {
    null = false
    type = numeric(24, 10)
  }
  column "effective_cost" {
    null = false
    type = numeric(24, 10)
  }
  column "list_cost" {
    null = true
    type = numeric(24, 10)
  }
  column "contracted_cost" {
    null = true
    type = numeric(24, 10)
  }
  column "pricing_quantity" {
    null = true
    type = numeric(24, 10)
  }
  column "pricing_unit" {
    null = true
    type = text
  }

  column "focus" {
    null    = true
    type    = jsonb
    comment = "the FOCUS long tail: Tags, SkuPriceDetails, CommitmentDiscount*, and all x_* custom columns"
  }
  column "fingerprint" {
    null    = false
    type    = text
    comment = "deterministic hash of the dimension tuple; the merge key within a bucket"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  # Idempotent target/bucket merge key. Source-native records also have a separate
  # source-scoped uniqueness constraint so corrections can move between targets/periods.
  # nulls_distinct = false keeps unmatched rows idempotent.
  index "config_costs_merge_uniq" {
    unique         = true
    nulls_distinct = false
    columns        = [column.source_key, column.config_id, column.period_start, column.period_end, column.fingerprint]
  }
  index "config_costs_source_record_uniq" {
    unique  = true
    columns = [column.source_key, column.source_record_id]
    where   = "source_record_id IS NOT NULL"
  }
  index "config_costs_period_brin_idx" {
    type    = BRIN
    columns = [column.period_start]
  }
  index "config_costs_config_period_idx" {
    columns = [column.config_id, column.period_start]
  }
  index "config_costs_unmatched_idx" {
    columns = [column.source_key, column.external_config_type, column.external_config_scraper_id, column.external_id]
    where   = "config_id IS NULL"
  }

  foreign_key "config_costs_config_fk" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_costs_scraper_fk" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_delete   = SET_NULL
  }

  check "config_costs_target" {
    expr = "config_id IS NOT NULL OR external_id IS NOT NULL"
  }
  check "config_costs_period" {
    expr = "period_end > period_start"
  }
  check "config_costs_grain" {
    expr = "grain IN ('day', 'week', 'month')"
  }
}
