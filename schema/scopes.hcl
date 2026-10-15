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
    null = false
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

  column "exclude" {
    null    = true
    type    = jsonb
    comment = "Array of scope exclusions, each shaped like a target. A resource matching one isn't in the Scope. NULL when there are none"
  }

  column "error" {
    null    = true
    type    = text
    comment = "why the scope isn't in effect, e.g. an agent it selects by doesn't exist. NULL when it's valid"
  }

  column "error_reason" {
    null    = true
    type    = text
    comment = "machine readable reason for error, e.g. AgentNotFound. NULL when it's valid"
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

  index "scopes_created_by_idx" {
    columns = [column.created_by]
  }

  foreign_key "scopes_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  check "scopes_namespace_not_empty" {
    expr = "namespace <> ''"
  }
}
