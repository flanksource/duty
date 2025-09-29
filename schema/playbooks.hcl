table "playbooks" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "namespace" {
    null    = false
    type    = text
    default = "default"
  }
  column "name" {
    null = false
    type = text
  }
  column "title" {
    null    = false
    type    = text
    default = ""
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
  column "category" {
    null    = false
    type    = text
    default = ""
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
  index "playbook_name_namespace_category_key" {
    unique  = true
    columns = [column.namespace, column.name, column.category]
    where   = "deleted_at IS NULL"
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
  column "spec" {
    null    = false
    type    = jsonb
    default = "{}" # temporary default value to make the migration possible. we can remove this later.
  }
  column "status" {
    null    = false
    type    = text
    default = "scheduled"
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
  column "timeout" {
    null    = false
    type    = bigint
    comment = "duration in nanoseconds"
    default = 1800000000000 # 30 minutes
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "notification_send_id" {
    null   = true
    column = "the notification dispatch that triggered this run"
    type   = uuid
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
  column "request" {
    null = true
    type = jsonb
  }
  column "agent_id" {
    null    = true
    default = var.uuid_nil
    type    = uuid
  }
  column "parent_id" {
    null    = true
    type    = uuid
    comment = "references the run that triggered this run"
  }
  column "error" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "playbook_run_parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [table.playbook_runs.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "playbook_run_playbook_id_fkey" {
    columns     = [column.playbook_id]
    ref_columns = [table.playbooks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "playbook_run_notification_send_id_fkey" {
    columns     = [column.notification_send_id]
    ref_columns = [table.notification_send_history.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
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
  index "idx_playbook_runs_parent_id" {
    columns = [column.parent_id]
  }
}

table "playbook_action_agent_data" {
  schema  = schema.public
  comment = "saves the necessary details for the agent to run a playbook action (eg: template env vars). Only applicable to agent runners."
  column "action_id" {
    null = false
    type = uuid
  }
  column "playbook_id" {
    comment = "saves the linked upstream playbook id"
    null    = false
    type    = uuid
  }
  column "run_id" {
    comment = "saves the linked upstream playbook run id"
    null    = false
    type    = uuid
  }
  column "spec" {
    comment = "Action spec provided by upstream"
    null    = false
    type    = jsonb
  }
  column "env" {
    comment = "templateEnv for the action provided by the upstream"
    null    = true
    type    = jsonb
  }
  foreign_key "playbook_action_template_env_agent_action_id_fkey" {
    columns     = [column.action_id]
    ref_columns = [table.playbook_run_actions.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
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
    type    = text
    default = "running"
  }
  column "retry_count" {
    null    = true
    type    = integer
    comment = "the nth retry of this action"
  }
  column "playbook_run_id" {
    null    = true
    type    = uuid
    comment = "a run id is mandatory except for an agent"
  }
  column "start_time" {
    null = true
    type = timestamptz
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
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  column "agent_id" {
    null    = true
    default = var.uuid_nil
    type    = uuid
    comment = "id of the agent that ran this action"
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
  foreign_key "playbook_run_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }

  index "playbook_run_actions_status_time_idx" {
    columns = [column.status, column.scheduled_time]
  }
  index "playbook_run_actions_playbook_run_id_idx" {
    columns = [column.playbook_run_id]
  }
}
