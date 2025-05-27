
table "config_backups" {
  schema = schema.public
  column "id" {
    null = false
    type = uuid
  }
  column "config_item_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null = false
    type = timestamptz
  }
  column "completed_at" {
    null = true
    type = timestamptz
  }
  column "status" {
    null = false
    type = text
  }
  column "size" {
    null    = false
    type    = bigint
    comment = "Size in bytes"
  }
  column "error" {
    null = true
    type = text
  }
  foreign_key "config_backups_config_item_id_fkey" {
    columns     = [column.config_item_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  primary_key {
    columns = [column.id]
  }
}

table "config_backup_restores" {
  schema = schema.public
  column "id" {
    null = false
    type = uuid
  }
  column "config_item_id" {
    null = false
    type = uuid
  }
  column "config_backup_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null = false
    type = timestamptz
  }
  column "completed_at" {
    null = true
    type = timestamptz
  }
  column "status" {
    null = false
    type = text
  }
  column "error" {
    null = true
    type = text
  }
  foreign_key "config_backup_restores_config_backup_id_fkey" {
    columns     = [column.config_backup_id]
    ref_columns = [table.config_backups.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_backup_restores_config_item_id_fkey" {
    columns     = [column.config_item_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  primary_key {
    columns = [column.id]
  }
}