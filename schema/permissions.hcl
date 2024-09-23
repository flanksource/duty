table "permissions" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "description" {
    null = false
    type = text
  }

  column "action" {
    null = false
    type = text
  }

  column "deny" {
    type = boolean
  }

  column "component_id" {
    null = true
    type = uuid
  }

  column "config_id" {
    null = true
    type = uuid
  }

  column "canary_id" {
    null = true
    type = uuid
  }

  column "playbook_id" {
    null = true
    type = uuid
  }

  column "connection_id" {
    null = true
    type = uuid
  }

  column "created_by" {
    null = false
    type = uuid
  }

  column "person_id" {
    null = true
    type = uuid
  }

  column "team_id" {
    null = true
    type = uuid
  }
  column "updated_by" {
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
  column "until" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "permissions_playbook_id_fkey" {
    columns     = [column.playbook_id]
    ref_columns = [table.playbooks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "permissions_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "permissions_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "permissions_connection_id_fkey" {
    columns     = [column.connection_id]
    ref_columns = [table.connections.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "permissions_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "permissions_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "permissions_person_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  foreign_key "permissions_team_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "permissions_config_id_idx" {
    columns = [column.config_id]
  }

  index "permissions_component_id_idx" {
    columns = [column.component_id]
  }
}
