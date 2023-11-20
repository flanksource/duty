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
  column "playbook_run_id" {
    null = true
    type = uuid
  }
  column "connection_id" {
    null    = true
    type    = uuid
    comment = "provides the credential to connect to the file store (S3, GCP, SFTP, ...)"
  }
  column "file_id" {
    null = false
    type = text
  }
  column "filename" {
    null = false
    type = text
  }
  column "size" {
    null = false
    type = text
  }
  column "checksum" {
    null = false
    type = text
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
  index "file_id_key" {
    unique  = true
    columns = [column.file_id]
  }
  foreign_key "artifacts_checks_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "artifacts_playbook_run_fkey" {
    columns     = [column.playbook_run_id]
    ref_columns = [table.playbook_runs.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
