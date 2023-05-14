table "canaries" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "agent_id" {
    null = true
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "namespace" {
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
    type = text
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
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "canaries_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "canaries_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "canaries_name_namespace_source_key" {
    unique  = true
    columns = [column.agent_id, column.name, column.namespace, column.source]
  }
}

table "check_statuses" {
  schema = schema.public
  column "check_id" {
    null = false
    type = uuid
  }
  column "details" {
    null = true
    type = jsonb
  }
  column "duration" {
    null = true
    type = integer
  }
  column "error" {
    null = true
    type = text
  }
  column "time" {
    null = false
    type = timestamptz
  }
  column "created_at" {
    null = false
    type = timestamptz
  }
  column "invalid" {
    null = true
    type = boolean
  }
  column "message" {
    null = true
    type = text
  }
  column "status" {
    null = true
    type = boolean
  }
  primary_key {
    columns = [column.check_id, column.time]
  }
  foreign_key "check_statuses_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "check_statuses_time_brin_idx" {
    type    = BRIN
    columns = [column.time]
  }
}

table "checks" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "canary_id" {
    null = false
    type = uuid
  }
  column "agent_id" {
    null = true
    type = uuid
  }
  column "type" {
    null = false
    type = text
  }
  column "name" {
    null = false
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "icon" {
    null = true
    type = text
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "labels" {
    null = true
    type = jsonb
  }
  column "owner" {
    null = true
    type = text
  }
  column "severity" {
    null = true
    type = text
  }
  column "category" {
    null = true
    type = text
  }
  column "transformed" {
    null = true
    type = boolean
  }
  column "last_runtime" {
    null = true
    type = timestamptz
  }
  column "last_transition_time" {
    null = true
    type = timestamptz
  }
  column "next_runtime" {
    null = true
    type = timestamptz
  }
  column "silenced_at" {
    null = true
    type = timestamptz
  }
  column "status" {
    null = true
    type = text
  }
  column "created_at" {
    null = true
    type = timestamptz
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
  foreign_key "checks_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "checks_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "checks_canary_id_type_name_key" {
    unique  = true
    columns = [column.canary_id, column.type, column.name]
  }
  index "checks_canary_id_transformed_idx" {
    columns = [column.canary_id, column.transformed]
  }
}

table "check_statuses_1h" {
  schema = schema.public
  column "check_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null = false
    type = timestamptz
  }
  column "duration" {
    null = false
    type = integer
  }
  column "total" {
    null = false
    type = integer
  }
  column "passed" {
    null = false
    type = integer
  }
  column "failed" {
    null = false
    type = integer
  }
  primary_key {
    columns = [column.check_id, column.created_at]
  }
  foreign_key "check_statuses_aggr_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "check_statuses_1h_created_at_brin_idx" {
    type    = BRIN
    columns = [column.created_at]
  }
}

table "check_statuses_1d" {
  schema = schema.public
  column "check_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null = false
    type = timestamptz
  }
  column "duration" {
    null = false
    type = integer
  }
  column "total" {
    null = false
    type = integer
  }
  column "passed" {
    null = false
    type = integer
  }
  column "failed" {
    null = false
    type = integer
  }
  primary_key {
    columns = [column.check_id, column.created_at]
  }
  foreign_key "check_statuses_aggr_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "check_statuses_1d_created_at_brin_idx" {
    type    = BRIN
    columns = [column.created_at]
  }
}
