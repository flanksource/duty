table "access_scopes" {
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

  column "person_id" {
    null    = true
    type    = uuid
    comment = "Person subject for this access scope"
  }

  column "team_id" {
    null    = true
    type    = uuid
    comment = "Team subject for this access scope"
  }

  column "resources" {
    null    = false
    type    = sql("text[]")
    comment = "Array of resource types this scope applies to"
  }

  column "scopes" {
    null    = false
    type    = jsonb
    comment = "Array of scope criteria (tags, agents, names) - OR logic between scopes, AND within"
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
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

  index "access_scopes_deleted_at_idx" {
    columns = [column.deleted_at]
  }

  index "access_scopes_namespace_name_idx" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  foreign_key "access_scopes_sub_person_id_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  foreign_key "access_scopes_sub_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  check "access_scopes_subject_check" {
    expr = "(person_id IS NOT NULL AND team_id IS NULL) OR (person_id IS NULL AND team_id IS NOT NULL)"
  }
}
