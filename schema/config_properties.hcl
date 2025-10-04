table "config_properties" {
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
    null = false
    type = uuid
  }
  column "label" {
    null = false
    type = text
  }
  column "tooltip" {
    null = true
    type = text
  }
  column "icon" {
    null = true
    type = text
  }
  column "type" {
    null = true
    type = text
  }
  column "color" {
    null = true
    type = text
  }
  column "order" {
    null = true
    type = int
  }
  column "text" {
    null = true
    type = text
  }
  column "value" {
    null = true
    type = numeric(16, 4)
  }
  column "unit" {
    null = true
    type = text
  }
  column "max" {
    null = true
    type = bigint
  }
  column "min" {
    null = true
    type = bigint
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
  foreign_key "config_properties_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "config_properties_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "config_properties_config_id_idx" {
    columns = [column.config_id]
  }
  index "config_properties_scraper_id_idx" {
    columns = [column.scraper_id]
  }
  check "config_property_text_or_value" {
    expr = "(text IS NOT NULL AND value IS NULL) OR (text IS NULL AND value IS NOT NULL)"
  }
}