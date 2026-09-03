table "config_costs" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "config_id" {
    null    = false
    type    = uuid
    comment = "always set; spend with no resource of its own is attributed to the scraper's root config item (AWS::::Account, GCP::Project, ...)"
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
  column "external_id" {
    null    = true
    type    = text
    comment = "FOCUS ResourceId as scraped, kept as provenance; the target is already resolved in config_id"
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
    comment = "level1 | level2 | level3. The width of each level is a property, so changing it is not a migration; each width must divide the next exactly."
  }

  # FOCUS dimensions kept queryable. Account identifiers live in focus; they are still
  # row identity via the fingerprint, they are just not filtered on directly.
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
    comment = "the FOCUS long tail plus billing_account_id / sub_account_id, and all x_* custom columns"
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

  # Idempotent bucket merge key.
  #
  # config_id is deliberately not part of it. The fingerprint already hashes the resource
  # the charge is for, so this tuple identifies the charge on its own, and config_id is
  # the attribution — which moves off the account root onto the resource as soon as the
  # catalog discovers it. Keying on config_id made that move insert a second row instead
  # of updating the first, leaving every late-discovered resource with a permanent
  # duplicate booked against its account.
  #
  # fingerprint precedes period_start so the compaction join, which seeks by source_key
  # and fingerprint over a range of period_start, still drives off the index.
  index "config_costs_merge_uniq" {
    unique  = true
    columns = [column.source_key, column.fingerprint, column.period_start, column.period_end]
  }
  index "config_costs_period_brin_idx" {
    type    = BRIN
    columns = [column.period_start]
  }
  # Drives both the read path (cost for a config over a window) and the compaction
  # passes, which select by grain and age.
  index "config_costs_config_period_idx" {
    columns = [column.config_id, column.period_start]
  }
  index "config_costs_grain_period_idx" {
    columns = [column.grain, column.period_end]
  }
  # Compaction selects the rows a provider has restated since the last pass. Without this
  # the window is a sequential scan of the whole retained history on every run.
  index "config_costs_updated_at_idx" {
    columns = [column.updated_at]
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

  check "config_costs_period" {
    expr = "period_end > period_start"
  }
  check "config_costs_grain" {
    expr = "grain IN ('level1', 'level2', 'level3')"
  }
}

# config_cost_compact is the queryable cost series, derived from config_costs and carrying
# every grain: 1h for recent data, 1d from the day threshold, 30d past the coarsening
# threshold. config_cost_summary and every read path go through this table; config_costs is
# the raw landing zone nothing reads.
#
# The shape is identical to config_costs so compaction is a plain INSERT ... SELECT and the
# same merge key keeps re-runs idempotent. Because config_costs retains its rows and
# providers restate open billing periods, compaction REPLACES a bucket rather than adding
# to it — re-running it must recompute the same total, not double it.
table "config_cost_compact" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "config_id" {
    null = false
    type = uuid
  }
  column "scraper_id" {
    null = true
    type = uuid
  }
  column "source_key" {
    null = false
    type = text
  }
  column "external_id" {
    null = true
    type = text
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

  column "period_start" {
    null = false
    type = timestamptz
  }
  column "period_end" {
    null = false
    type = timestamptz
  }
  column "grain" {
    null = false
    type = text
  }

  column "charge_category" {
    null = false
    type = text
  }
  column "charge_class" {
    null = true
    type = text
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
  column "billing_currency" {
    null = false
    type = text
  }

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
    null = true
    type = jsonb
  }
  column "fingerprint" {
    null = false
    type = text
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

  # Identical to config_costs_merge_uniq, and for the same reason: config_id is the
  # attribution a charge currently carries, not part of what makes it that charge.
  index "config_cost_compact_merge_uniq" {
    unique  = true
    columns = [column.source_key, column.fingerprint, column.period_start, column.period_end]
  }
  index "config_cost_compact_period_brin_idx" {
    type    = BRIN
    columns = [column.period_start]
  }
  index "config_cost_compact_config_period_idx" {
    columns = [column.config_id, column.period_start]
  }
  # Both the superseded-delete and the compaction pass that supersedes those rows select
  # by grain and age, the same way config_costs is read.
  index "config_cost_compact_grain_period_idx" {
    columns = [column.grain, column.period_end]
  }

  foreign_key "config_cost_compact_config_fk" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_cost_compact_scraper_fk" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_delete   = SET_NULL
  }

  check "config_cost_compact_period" {
    expr = "period_end > period_start"
  }
  check "config_cost_compact_grain" {
    expr = "grain IN ('level1', 'level2', 'level3')"
  }
}
