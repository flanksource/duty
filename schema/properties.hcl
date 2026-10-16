table "properties" {
  schema = schema.public
  column "name" {
    null = false
    type = text
  }
  column "value" {
    null = false
    type = text
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "created_at" {
    null = true
    type = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null = true
    type = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.name]
  }
  index "properties_created_by_idx" {
    columns = [column.created_by]
  }
  foreign_key "properties_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
