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
  column "config_change_id" {
    null = true
    type = uuid
  }
  column "scraper_id" {
    null    = true
    type    = uuid
    comment = "durable owner for scraper-generated artifacts; references config_scrapers"
  }
  column "job_history_id" {
    null    = true
    type    = uuid
    comment = "creator/provenance job run for scraper-generated artifacts; set null when job_history rows are pruned"
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
  column "content" {
    null    = true
    type    = bytea
    comment = "inline blob storage for artifact data when no external connection is configured"
  }
  column "compression_type" {
    null    = true
    type    = text
    comment = "compression algorithm applied to content: gzip, zstd, or none"
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
  foreign_key "artifacts_config_change_fkey" {
    columns     = [column.config_change_id]
    ref_columns = [table.config_changes.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "artifacts_scraper_fkey" {
    columns     = [column.scraper_id]
    ref_columns = [table.config_scrapers.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "artifacts_job_history_fkey" {
    columns     = [column.job_history_id]
    ref_columns = [table.job_history.column.id]
    on_update   = NO_ACTION
    on_delete   = SET_NULL
  }
  index "artifacts_check_id_idx" {
    columns = [column.check_id]
  }
  index "artifacts_playbook_run_action_id_idx" {
    columns = [column.playbook_run_action_id]
  }
  index "artifacts_config_change_id_idx" {
    columns = [column.config_change_id]
  }
  index "artifacts_scraper_id_idx" {
    columns = [column.scraper_id]
  }
  index "artifacts_job_history_id_idx" {
    columns = [column.job_history_id]
  }
}
