table "config_locations" {
  schema = schema.public
  column "id" {
    null = false
    type = uuid
  }
  column "location" {
    null = false
    type = text
  }
  foreign_key "config_locations_config_item_id_fkey" {
    columns     = [column.id]
    ref_columns = [table.config_items.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  # Index is created in views/041_config_location_functions.sql
  # because atlas doesn't support creating index with text_pattern_ops
  index "config_locations_config_item_id_location_key" {
    unique  = true
    columns = [column.id, column.location]
  }
}