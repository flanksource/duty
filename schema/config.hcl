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
  column "scraper_id" {
    null = true
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
    null    = true
    type    = text
    default = "open"
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
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
    comment = "is_pushed when set to true indicates that the config analysis has been pushed to upstream."
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_analysis_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "config_analysis_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_analysis_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }

  index "config_analysis_config_id_idx" {
    columns = [column.config_id]
  }
  index "config_analysis_config_id_status_open_idx" {
    columns = [column.config_id]
    where   = "status = 'open'"
  }
  index "config_analysis_last_observed_idx" {
    type    = BRIN
    columns = [column.last_observed]
  }
  index "config_analysis_scraper_id_idx" {
    columns = [column.scraper_id]
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
  column "diff" {
    null = true
    type = text
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

  column "count" {
    null    = false
    default = 1
    type    = int
  }

  column "fingerprint" {
    null = true
    type = text
  }

  column "first_observed" {
    null    = false
    default = sql("now()")
    type    = timestamptz
  }

  column "is_pushed" {
    null    = false
    default = false
    type    = bool
    comment = "is_pushed when set to true indicates that the config changes has been pushed to upstream."
  }
  column "inserted_at" {
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
    on_delete   = CASCADE
  }

  index "config_changes_created_at_brin_idx" {
    type    = BRIN
    columns = [column.created_at]
  }
  index "config_changes_config_id_external_change_id_key" {
    unique  = true
    columns = [column.config_id, column.external_change_id]
  }
  index "config_changes_change_type_idx" {
    columns = [column.change_type]
  }
  index "config_changes_config_id_idx" {
    columns = [column.config_id]
  }
  index "config_changes_config_id_change_type_idx" {
    columns = [column.config_id, column.change_type]
  }
  index "config_changes_fingerprint_created_at_idx" {
    columns = [column.fingerprint, column.created_at]
  }
  index "config_changes_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
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
    null    = false
    default = var.uuid_nil
    type    = uuid
  }
  column "icon" {
    null = true
    type = text
  }
  column "scraper_id" {
    null = true
    type = uuid
  }
  column "config_class" {
    null = false
    type = text
  }
  column "status" {
    null = true
    type = text
  }
  column "health" {
    null = true
    type = text
  }
  column "ready" {
    null = true
    type = bool
  }
  column "external_id" {
    null = true
    type = sql("text[]")
  }
  column "type" {
    null = false
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
  column "description" {
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
  column "labels" {
    null = true
    type = jsonb
  }
  column "tags" {
    null    = true
    type    = jsonb
    comment = "contains a list of tags"
  }
  column "tags_values" {
    null    = true
    type    = jsonb
    comment = "derived from the tags column to enhance search performance for unkeyed tag values"
    as {
      # without the ::jsonpath type casting
      # we get this error from Atlas: failed to compute diff: failed to diff realms:
      # changing the generation expression for a column "tags_values" is not supported
      expr = "(jsonb_path_query_array(tags, '$[*].*'::jsonpath))"
      type = STORED
    }
  }
  column "properties" {
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
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
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
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  column "delete_reason" {
    null = true
    type = text
  }
  column "inserted_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
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
  index "config_items_path_is_pushed_idx" {
    on {
      expr = "length(path)"
    }
    where = "is_pushed IS FALSE"
  }
  index "idx_config_items_scraper_id" {
    columns = [column.scraper_id]
  }
  index "idx_config_items_parent_id" {
    columns = [column.parent_id]
  }
  index "idx_config_items_external_id" {
    columns = [column.external_id]
    type    = GIN
  }
  index "idx_config_items_deleted_at" {
    columns = [column.deleted_at]
  }
  index "idx_config_items_tags" {
    columns = [column.tags]
    type    = GIN
  }
  index "idx_config_items_tags_values" {
    columns = [column.tags_values]
    type    = GIN
  }
  index "idx_config_items_name" {
    columns = [column.agent_id, column.name, column.type, column.config_class]
  }
  index "idx_config_items_scraper_id_deleted_at_null" {
    columns = [column.scraper_id]
    where   = "deleted_at IS NULL"
  }
  index "idx_config_items_path" {
    columns = [column.path]
  }
  check "config_item_name_type_not_empty" {
    expr = "LENGTH(name) > 0 AND LENGTH(type) > 0"
  }
}

table "config_items_last_scraped_time" {
  schema = schema.public
  unlogged = true
  column "config_id" {
    null = false
    type = uuid
  }
  column "last_scraped_time" {
    null = false
    type = timestamptz
    default = sql("now()")
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
    comment = "is_pushed when set to true indicates that the config analysis has been pushed to upstream."
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
    columns = [column.config_id]
  }
  foreign_key "config_items_last_scraped_at_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
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
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  foreign_key "config_relationships_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "config_relationships_related_id_fkey" {
    columns     = [column.related_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "config_relationships_related_id_config_id_relation_key" {
    unique  = true
    columns = [column.related_id, column.config_id, column.relation]
  }
  index "idx_config_relationships_deleted_at" {
    columns = [column.deleted_at]
  }
  index "config_relationships_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
}

table "check_config_relationships" {
  schema = schema.public
  column "config_id" {
    null = false
    type = uuid
  }
  column "check_id" {
    null = false
    type = uuid
  }
  column "canary_id" {
    null = false
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
  column "selector_id" {
    null = true
    type = text
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  foreign_key "check_config_relationships_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "check_config_relationships_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "check_config_relationships_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "check_config_relationships_config_id_check_id_canary__key" {
    unique  = true
    columns = [column.config_id, column.check_id, column.canary_id, column.selector_id]
  }
  index "check_config_relationships_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
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
  column "namespace" {
    null = true
    type = text
  }
  column "spec" {
    null = false
    type = jsonb
  }
  column "source" {
    null = false
    type = enum.source
  }
  column "application_id" {
    null    = true
    type    = uuid
    comment = "application that generated this scraper"
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "agent_id" {
    null    = false
    default = var.uuid_nil
    type    = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null = true
    type = timestamptz
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_scrapers_application_id_fkey" {
    columns     = [column.application_id]
    ref_columns = [table.applications.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "config_scrapers_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_scrapers_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_scrapers_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
}

table "scrape_plugins" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "name" {
    type = text
  }
  column "namespace" {
    null = true
    type = text
  }
  column "spec" {
    null = false
    type = jsonb
  }
  column "source" {
    null = false
    type = enum.source
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
    null    = true
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
  foreign_key "config_scraper_plugins_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
