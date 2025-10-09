table "scope_bindings" {
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

  column "persons" {
    null    = true
    type    = sql("text[]")
    default = sql("'{}'::text[]")
    comment = "Array of person emails"
  }

  column "teams" {
    null    = true
    type    = sql("text[]")
    default = sql("'{}'::text[]")
    comment = "Array of team names"
  }

  column "scopes" {
    null    = false
    type    = sql("text[]")
    comment = "Array of scope names (must be in same namespace)"
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

  index "scope_bindings_deleted_at_idx" {
    columns = [column.deleted_at]
  }

  index "scope_bindings_namespace_name_idx" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  foreign_key "scope_bindings_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  check "scope_bindings_subjects_check" {
    expr = "array_length(persons, 1) > 0 OR array_length(teams, 1) > 0"
  }
}
