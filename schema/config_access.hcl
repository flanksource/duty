table "external_users" {
  schema = schema.public
  column "id" {
    null    = false
    type = uuid
    default = sql("generate_ulid()")
  }
  column "account_id" {
    comment = "Azure tenant ID, AWS account ID, GCP project ID"
    type    = text
  }
  column "aliases" {
    type = sql("text[]")
    null = true
  }
  column "name" {
    type = text
  }
  column "user_type" {
    type = text
  }
  column "email" {
    type = text
    null = true
  }
  column "scraper_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    type = timestamptz
  }
  column "updated_at" {
    type = timestamptz
    null = true
  }
  column "deleted_at" {
    type = timestamptz
    null = true
  }
  column "created_by" {
    type = uuid
    null = true
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "external_users_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "external_groups" {
  schema = schema.public
  column "id" {
    null    = false
    type = uuid
    default = sql("generate_ulid()")
  }
  column "account_id" {
    comment = "Azure tenant ID, AWS account ID, GCP project ID"
    type    = text
  }
  column "scraper_id" {
    null = false
    type = uuid
  }
  column "aliases" {
    type = sql("text[]")
    null = true
  }
  column "name" {
    type = text
  }
  column "group_type" {
    type = text
  }
  column "created_at" {
    type = timestamptz
  }
  column "updated_at" {
    type = timestamptz
    null = true
  }
  column "deleted_at" {
    type = timestamptz
    null = true
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "external_groups_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "external_user_groups" {
  schema = schema.public
  column "external_user_id" {
    type = uuid
  }
  column "external_group_id" {
    type = uuid
  }
  column "deleted_at" {
    type = timestamptz
    null = true
  }
  column "deleted_by" {
    type = uuid
    null = true
  }
  column "created_at" {
    type = timestamptz
  }
  column "created_by" {
    type = uuid
    null = true
  }
  primary_key {
    columns = [column.external_user_id, column.external_group_id]
  }
  foreign_key "external_user_fk" {
    columns     = [column.external_user_id]
    ref_columns = [table.external_users.column.id]
    on_delete   = NO_ACTION
  }
  foreign_key "external_group_fk" {
    columns     = [column.external_group_id]
    ref_columns = [table.external_groups.column.id]
    on_delete   = NO_ACTION
  }
}

table "external_roles" {
  schema = schema.public
  column "id" {
    null    = false
    type = uuid
    default = sql("generate_ulid()")

  }
  column "scraper_id" {
    type = uuid
    null = true
  }
  column "application_id" {
    null = true
    type = uuid
  }
  column "account_id" {
    comment = "Azure tenant ID, AWS account ID, GCP project ID"
    type    = text
  }
  column "aliases" {
    type = sql("text[]")
    null = true
  }
  column "role_type" {
    type = text
  }
  column "name" {
    type = text
  }
  column "description" {
    type = text
    null = true
  }
  primary_key {
    columns = [column.id]
  }
  check "external_roles_parent" {
    comment = "external roles can be created from application mapping or from scraper"
    expr    = "application_id IS NOT NULL OR scraper_id IS NOT NULL"
  }
  foreign_key "external_roles_application_id_fkey" {
    columns     = [column.application_id]
    ref_columns = [table.applications.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "access_reviews" {
  schema = schema.public
  column "id" {
    type = uuid
  }
  column "scraper_id" {
    null = false
    type = uuid
  }
  column "aliases" {
    type = text
    null = true
  }
  column "config_id" {
    type = uuid
  }
  column "external_user_id" {
    type = uuid
  }
  column "external_role_id" {
    type = uuid
  }
  column "created_at" {
    type = timestamptz
  }
  column "created_by" {
    type = uuid
    null = true
  }
  column "source" {
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "config_fk" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_delete   = CASCADE
  }
  foreign_key "external_user_fk" {
    columns     = [column.external_user_id]
    ref_columns = [table.external_users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "external_role_fk" {
    columns     = [column.external_role_id]
    ref_columns = [table.external_roles.column.id]
    on_delete   = CASCADE
  }
  foreign_key "access_reviews_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "config_access" {
  schema = schema.public
  column "id" {
    type    = text
    comment = "not a uuid. depends on the source. example: Microsoft has 0Tr1liTQeU2nA2LDmGCS4qwxw-A6_GhNos_LscLVs6w"
  }
  column "config_id" {
    type = uuid
  }
  column "scraper_id" {
    type = uuid
    null = true
  }
  column "application_id" {
    null = true
    type = uuid
  }
  column "external_user_id" {
    type = uuid
    null = true
  }
  column "external_group_id" {
    type = uuid
    null = true
  }
  column "external_role_id" {
    type = uuid
    null = true
  }
  column "created_at" {
    type = timestamptz
  }
  column "deleted_at" {
    type = timestamptz
    null = true
  }
  column "deleted_by" {
    type = uuid
    null = true
  }
  column "last_reviewed_at" {
    type = timestamptz
    null = true
  }
  column "last_reviewed_by" {
    type = uuid
    null = true
  }
  column "created_by" {
    type = uuid
    null = true
  }
  primary_key {
    columns = [column.id]
  }
  index "config_access_config_id_external_user_id_external_group_id_external_role_id_key" {
    unique  = true
    columns = [column.config_id, column.external_user_id, column.external_group_id, column.external_role_id]
    where   = "deleted_at IS NULL"
  }
  foreign_key "config_fk" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_delete   = CASCADE
  }
  foreign_key "external_user_fk" {
    columns     = [column.external_user_id]
    ref_columns = [table.external_users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "external_group_fk" {
    columns     = [column.external_group_id]
    ref_columns = [table.external_groups.column.id]
    on_delete   = CASCADE
  }
  foreign_key "external_role_fk" {
    columns     = [column.external_role_id]
    ref_columns = [table.external_roles.column.id]
    on_delete   = CASCADE
  }
  foreign_key "config_access_application_id_fkey" {
    columns     = [column.application_id]
    ref_columns = [table.applications.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_access_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  check "config_access_parent" {
    expr    = "scraper_id IS NOT NULL OR application_id IS NOT NULL"
    comment = "config access can be created from role mapping in application or from scraper"
  }
  check "at_least_one_id" {
    expr = "external_user_id IS NOT NULL OR external_group_id IS NOT NULL OR external_role_id IS NOT NULL"
  }
}

table "config_access_logs" {
  schema = schema.public
  column "external_user_id" {
    null = false
    type = uuid
  }
  column "config_id" {
    null = false
    type = uuid
  }
  column "scraper_id" {
    null = false
    type = uuid
  }
  column "mfa" {
    type = boolean
    null = true
  }
  column "properties" {
    type = jsonb
    null = true
  }
  column "created_at" {
    type = timestamptz
  }
  primary_key {
    columns = [column.config_id, column.external_user_id, column.scraper_id]
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
  foreign_key "config_access_logs_scraper_id_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
