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
  column "labels" {
    null = true
    type = jsonb
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
    comment = "Deprecated:The last time the view queries were run and persisted"
  }
  column "error" {
    null = true
    type = text
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
  index "views_created_by_idx" {
    columns = [column.created_by]
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
    null    = false
    type    = uuid
    comment = "maps one-to-one with views.id"
  }
  column "request_fingerprint" {
    null    = false
    type    = text
    default = ""
    comment = "Fingerprint of request variables for cache differentiation"
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
  column "refreshed_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
    comment = "Last time this view was refreshed for this request fingerprint"
  }
  foreign_key "view_panels_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  primary_key {
    columns = [column.view_id, column.request_fingerprint]
    comment = "one record per view per request fingerprint"
  }
  index "view_panels_agent_id_idx" {
    columns = [column.agent_id]
  }
  index "idx_view_panels_request_fingerprint" {
    columns = [column.view_id, column.request_fingerprint]
  }
}