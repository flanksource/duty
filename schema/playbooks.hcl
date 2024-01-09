table "playbooks" {
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
  column "icon" {
    null = true
    type = text
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
  column "source" {
    null = false
    type = enum.source
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
  index "playbook_name_key" {
    unique  = true
    columns = [column.name]
  }
  foreign_key "playbook_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "playbook_approvals" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "run_id" {
    null = false
    type = uuid
  }
  column "person_id" {
    null = true
    type = uuid
  }
  column "team_id" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  foreign_key "playbook_approval_run_id_fkey" {
    columns     = [column.run_id]
    ref_columns = [table.playbook_runs.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_approval_person_approver_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_approval_team_approver_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  comment = "Keeps track of approvals on a playbook run"
}

enum "playbook_run_status" {
  schema = schema.public
  values = ["scheduled", "running", "cancelled", "completed", "failed", "pending", "sleeping"]
}

enum "playbook_action_status" {
  schema = schema.public
  values = ["completed", "running", "failed", "sleeping", "skipped"]
}

table "playbook_runs" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "playbook_id" {
    null = false
    type = uuid
  }
  column "status" {
    null    = false
    type    = enum.playbook_run_status
    default = "pending"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "start_time" {
    null    = true
    type    = timestamptz
    comment = "the time the run was first started"
  }
  column "scheduled_time" {
    null    = false
    type    = timestamptz
    default = sql("now()")
    comment = "the time the run is supposed to start/resume"
  }
  column "end_time" {
    null = true
    type = timestamptz
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "check_id" {
    null = true
    type = uuid
  }
  column "config_id" {
    null = true
    type = uuid
  }
  column "component_id" {
    null = true
    type = uuid
  }
  column "parameters" {
    null = true
    type = jsonb
  }
  column "agent_id" {
    null = true
    type = uuid
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "playbook_run_playbook_id_fkey" {
    columns     = [column.playbook_id]
    ref_columns = [table.playbooks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "playbook_run_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_run_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_run_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_run_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_run_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  check "check_component_or_config_or_check" {
    expr    = <<EOF
    (component_id IS NOT NULL AND config_id IS NULL AND check_id IS NULL) OR
    (component_id IS NULL AND config_id IS NOT NULL AND check_id IS NULL) OR
    (component_id IS NULL AND config_id IS NULL AND check_id IS NOT NULL)
    EOF
    comment = "Need exactly one of the following: check id, component id, or config id."
  }
}

table "playbook_run_actions" {
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
  column "status" {
    null    = false
    type    = enum.playbook_action_status
    default = "running"
  }
  column "playbook_run_id" {
    null = false
    type = uuid
  }
  column "start_time" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "scheduled_time" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "end_time" {
    null = true
    type = timestamptz
  }
  column "result" {
    null = true
    type = jsonb
  }
  column "error" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "playbook_run_playbook_run_id_fkey" {
    columns     = [column.playbook_run_id]
    ref_columns = [table.playbook_runs.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
}
