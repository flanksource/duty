table "views" {
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
  column "spec" {
    null = false
    type = jsonb
  }
  column "source" {
    null    = false
    type    = enum.source
    default = "KubernetesCRD"
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
  column "last_ran" {
    null    = true
    type    = timestamptz
    comment = "The last time the view queries were run and persisted"
  }
  column "error" {
    null    = true
    type    = text
  }
  column "updated_at" {
    null = true
    type = timestamptz
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "views_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "view_panels" {
  schema = schema.public
  column "view_id" {
    null = false
    type = uuid
  }
  column "results" {
    null = false
    type = jsonb
  }
  column "agent_id" {
    null    = false
    default = var.uuid_nil
    type    = uuid
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  foreign_key "view_panels_view_id_fkey" {
    columns     = [column.view_id]
    ref_columns = [table.views.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "view_panels_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  primary_key {
    columns = [column.view_id]
    comment = "one record per view"
  }
}