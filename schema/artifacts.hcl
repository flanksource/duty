table "artifacts" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "check_id" {
    null = true
    type = uuid
  }
  column "check_time" {
    null = true
    type = timestamptz
  }
  column "playbook_run_action_id" {
    null = true
    type = uuid
  }
  column "connection_id" {
    null    = true
    type    = uuid
    comment = "provides the credential to connect to the file store (S3, GCS, SMB, SFTP, ...)"
  }
  column "path" {
    null = false
    type = text
  }
  column "filename" {
    null = false
    type = text
  }
  column "content_type" {
    null = true
    type = text
  }
  column "size" {
    null = false
    type = integer
  }
  column "checksum" {
    null = false
    type = text
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
    comment = "indicates whether the artifact record has been pushed to upstream."
  }
  column "is_data_pushed" {
    null    = false
    default = false
    type    = bool
    comment = "indicates whether the artifact data itself has been pushed to upstream."
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
  column "expires_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "artifacts_checks_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "artifacts_playbook_run_action_fkey" {
    columns     = [column.playbook_run_action_id]
    ref_columns = [table.playbook_run_actions.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "artifacts_check_id_idx" {
    columns = [column.check_id]
  }
  index "artifacts_playbook_run_action_id_idx" {
    columns = [column.playbook_run_action_id]
  }
}
