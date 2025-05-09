table "external_users" {
  schema = schema.public
  column "id" {
    type = uuid
  }
  column "aliases" {
    type = text
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
}

table "external_groups" {
  schema = schema.public
  column "id" {
    type = uuid
  }
  column "aliases" {
    type = text
    null = true
  }
  column "name" {
    type = text
  }
  column "group_type" {
    type = text
  }
  primary_key {
    columns = [column.id]
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
    type = uuid
  }
  column "aliases" {
    type = text
    null = true
  }
  column "role_type" {
    type = text
  }
  column "name" {
    type = text
  }
  column "spec" {
    type = jsonb
    null = true
  }
  column "description" {
    type = text
    null = true
  }
  primary_key {
    columns = [column.id]
  }
}

table "access_reviews" {
  schema = schema.public
  column "id" {
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
}

table "config_access" {
  schema = schema.public
  column "id" {
    type = uuid
  }
  column "config_id" {
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
}
