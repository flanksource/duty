table "config_access_logs" {
  schema = schema.public
  column "id" {
    null    = false
    type    = text
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
  column "external_user_id" {
    null = false
    type = uuid
  }
  column "external_role_id" {
    null    = false
    type    = uuid
    default = "00000000-0000-0000-0000-000000000000"
  }
  column "client_ip" {
    null    = false
    type    = text
    default = ""
  }
  column "verb" {
    null = true
    type = text
  }
  column "outcome" {
    null    = false
    type    = text
    default = "allowed"
  }
  column "mfa" {
    type    = boolean
    null    = false
    default = false
  }
  column "properties" {
    type = jsonb
    null = true
  }
  column "count" {
    type    = integer
    default = 1
  }
  column "fingerprint" {
    null    = false
    type    = text
    default = ""
  }
  column "first_observed" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    type = timestamptz
  }
  column "inserted_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "bucket_start" {
    null    = false
    type    = timestamptz
    default = sql("date_trunc('day', now())")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_access_logs_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_access_logs_external_user_id_fkey" {
    columns     = [column.external_user_id]
    ref_columns = [table.external_users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_access_logs_external_role_id_fkey" {
    columns     = [column.external_role_id]
    ref_columns = [table.external_roles.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_access_logs_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_access_logs_external_user_id_idx" {
    columns = [column.external_user_id]
  }
  index "config_access_logs_scraper_id_idx" {
    columns = [column.scraper_id]
  }
  index "config_access_logs_dedupe_idx" {
    columns = [column.config_id, column.scraper_id, column.client_ip, column.external_role_id, column.external_user_id, column.outcome, column.mfa, column.bucket_start]
    unique  = true
  }
}
