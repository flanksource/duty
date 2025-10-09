table "scopes" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }

  column "name" {
    null = false
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

  column "targets" {
    null    = false
    type    = jsonb
    comment = "Array of scope targets - each target contains one resource type (config/component/playbook/canary/*) with selector"
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
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

  index "scopes_deleted_at_idx" {
    columns = [column.deleted_at]
  }

  index "scopes_namespace_name_idx" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  foreign_key "scopes_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
