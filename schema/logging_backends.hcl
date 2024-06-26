table "logging_backends" {
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
  column "labels" {
    null = true
    type = jsonb
  }
  column "spec" {
    null = false
    type = jsonb
  }
  column "source" {
    null = true
    type = enum.source
  }
  column "agent_id" {
    null = true
    type = uuid
  }
  column "created_at" {
    null = true
    type = timestamptz
    default = sql("now()")
  }
  column "created_by" {
    null = true
    type = uuid
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
  foreign_key "logging_backends_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
