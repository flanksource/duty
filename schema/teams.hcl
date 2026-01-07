schema "public" {
}

table "people" {
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
  column "avatar" {
    null = true
    type = text
  }
  column "type" {
    null = true
    type = text
  }
  column "team_id" {
    null = true
    type = uuid
  }
  column "organization" {
    null = true
    type = text
  }
  column "title" {
    null = true
    type = text
  }
  column "email" {
    null = true
    type = text
  }
  column "phone" {
    null = true
    type = text
  }
  column "properties" {
    null = true
    type = jsonb
  }
  column "external_id" {
    null = true
    type = text
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
  column "last_login" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  index "people_email_unique_idx" {
    unique  = true
    columns = [column.email]
    where   = "deleted_at IS NULL"
  }
  index "people_external_id_idx" {
    columns = [column.external_id]
  }
  foreign_key "people_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "team_members" {
  schema = schema.public
  column "team_id" {
    null = false
    type = uuid
  }
  column "person_id" {
    null = false
    type = uuid
  }
  column "source" {
    type = text
    null = false
    default = "UI"
  }
  primary_key {
    columns = [column.team_id, column.person_id]
  }
  foreign_key "team_members_person_id_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "team_members_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
}

table "teams" {
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
  column "icon" {
    null = true
    type = text
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "source" {
    null = true
    type = text
  }
  column "created_by" {
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
  primary_key {
    columns = [column.id]
  }
  index "team_name_deleted_at_key" {
    unique  = true
    columns = [column.name, column.deleted_at]
  }
  foreign_key "teams_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "team_components" {
  schema = schema.public
  column "team_id" {
    null = false
    type = uuid
  }
  column "component_id" {
    null = false
    type = uuid
  }
  column "role" {
    null = true
    type = text
  }
  column "selector_id" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.team_id, column.component_id]
  }
  foreign_key "team_components_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "team_components_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "team_components_team_id_component_id_selector_id_key" {
    unique  = true
    columns = [column.team_id, column.component_id, column.selector_id]
  }
}

table "saved_query" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "icon" {
    null = true
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "query" {
    null = false
    type = text
  }
  column "columns" {
    null = true
    type = jsonb
  }
  column "created_by" {
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
  primary_key {
    columns = [column.id]
  }
}
