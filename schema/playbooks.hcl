table "playbooks" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "description" {
    null = true
    type = text
  }
  column "spec" {
    null = false
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
  foreign_key "topologies_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "playbook_runs" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "created_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "playbook_id" {
    null = false
    type = uuid
  }
  column "duration" {
    null = true
    type = integer
  }
  column "result" {
    null = false
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "playbook_runs_playbook_id_fkey" {
    columns     = [column.playbook_id]
    ref_columns = [table.playbooks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
}