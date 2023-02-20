table "push_queue" {
  schema = schema.public
  column "id" {
    null = false
    type = uuid
    default = sql("generate_ulid()")
  }
  column "item_id" {
    null = false
    type = text # not a uuid. it could also be a composite key.
  }
  column "table_name" {
    null    = false
    type    = text
  }
  column "operation" {
    null    = false
    type    = text
  }
  column "created_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  index "push_queue_table_name_key" {
    columns = [column.table_name]
  }
}