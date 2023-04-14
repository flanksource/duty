table "config_analysis" {
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
  column "created_by" {
    null = true
    type = uuid
  }
  column "source" {
    null = true
    type = text
  }
  column "analyzer" {
    null = false
    type = text
  }
  column "analysis_type" {
    null = true
    type = text
  }
  column "severity" {
    null = true
    type = text
  }
  column "summary" {
    null = true
    type = text
  }
  column "status" {
    null = true
    type = text
  }
  column "message" {
    null = true
    type = text
  }
  column "analysis" {
    null = true
    type = jsonb
  }
  column "first_observed" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "last_observed" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_analysis_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_analysis_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "config_changes" {
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
  column "external_change_id" {
    null = true
    type = text
  }
  column "external_created_by" {
    null = true
    type = text
  }
  column "change_type" {
    null = true
    type = text
  }
  column "severity" {
    null = true
    type = text
  }
  column "source" {
    null = true
    type = text
  }
  column "summary" {
    null = true
    type = text
  }
  column "patches" {
    null = true
    type = jsonb
  }
  column "details" {
    null = true
    type = jsonb
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_changes_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_changes_config_id_external_change_id_key" {
    unique  = true
    columns = [column.config_id, column.external_change_id]
  }
}

table "config_items" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "agent_id" {
    null = true
    type = uuid
  }
  column "icon" {
    null = true
    type = text
  }
  column "scraper_id" {
    null = true
    type = uuid
  }
  column "config_type" {
    null = false
    type = text
  }
  column "external_id" {
    null = true
    type = sql("text[]")
  }
  column "external_type" {
    null = true
    type = text
  }
  column "cost_per_minute" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_1d" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_7d" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_30d" {
    null = true
    type = numeric(16, 4)
  }
  column "name" {
    null = true
    type = text
  }
  column "namespace" {
    null = true
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "account" {
    null = true
    type = text
  }
  column "config" {
    null = true
    type = jsonb
  }
  column "source" {
    null = true
    type = text
  }
  column "tags" {
    null = true
    type = jsonb
  }
  column "parent_id" {
    null = true
    type = uuid
  }
  column "path" {
    null = true
    type = text
  }
  column "created_by" {
    null = true
    type = uuid
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
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_items_parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_items_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_items_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_config_items_external_id" {
    columns = [column.external_id]
    type    = GIN
  }
}

table "config_relationships" {
  schema = schema.public
  column "config_id" {
    null = false
    type = uuid
  }
  column "related_id" {
    null = false
    type = uuid
  }
  column "relation" {
    null = true
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
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  column "selector_id" {
    null = true
    type = text
  }
  foreign_key "config_relationships_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_relationships_related_id_fkey" {
    columns     = [column.related_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_relationships_related_id_config_id_selector_id_key" {
    unique  = true
    columns = [column.related_id, column.config_id, column.selector_id]
  }
}

table "config_scrapers" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "description" {
    null = true
    type = text
  }
  column "name" {
    type = text
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "created_by" {
    null = true
    type = uuid
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
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_scrapers_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

}
