table "topologies" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "agent_id" {
    null    = false
    default = var.uuid_nil
    type    = uuid
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
  column "source" {
    null    = false
    type    = enum.source
    default = "UI"
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "created_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  column "schedule" {
    null = true
    type = text
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "topologies_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "topologies_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "topologies_name_namespace_key" {
    unique  = true
    columns = [column.agent_id, column.name, column.namespace]
    where   = "deleted_at IS NULL AND agent_id = '00000000-0000-0000-0000-000000000000'"
  }
  index "topologies_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
  index "topologies_agent_id_idx" {
    columns = [column.agent_id]
  }
  index "topologies_created_by_idx" {
    columns = [column.created_by]
  }
}


table "component_relationships" {
  schema = schema.public
  column "component_id" {
    null = false
    type = uuid
  }
  column "relationship_id" {
    null = false
    type = uuid
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
  column "selector_id" {
    null = true
    type = text
  }
  column "relationship_path" {
    null = true
    type = text
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }

  foreign_key "component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "component_relationships_relationship_id_fkey" {
    columns     = [column.relationship_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "component_relationships_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
  index "component_relationships_component_id_relationship_id_select_key" {
    unique  = true
    columns = [column.component_id, column.relationship_id, column.selector_id]
  }
  index "idx_component_relationships_deleted_at" {
    columns = [column.deleted_at]
  }
  index "component_relationships_relationship_id_idx" {
    columns = [column.relationship_id]
  }
}

table "components" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "agent_id" {
    null    = false
    default = var.uuid_nil
    type    = uuid
  }
  column "topology_id" {
    null = true
    type = uuid
  }
  column "external_id" {
    null = false
    type = text
  }
  column "parent_id" {
    null = true
    type = uuid
  }
  column "config_id" {
    null = true
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "text" {
    null = true
    type = text
  }
  column "topology_type" {
    null = true
    type = text
  }
  column "namespace" {
    null = true
    type = text
  }
  column "labels" {
    null = true
    type = jsonb
  }
  column "hidden" {
    null    = false
    type    = boolean
    default = false
  }
  column "silenced" {
    null    = false
    type    = boolean
    default = false
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  column "status" {
    null = false
    type = text
  }
  column "status_expr" {
    null = true
    type = text
  }
  column "health_expr" {
    null = true
    type = text
  }
  column "health" {
    null = true
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "lifecycle" {
    null = true
    type = text
  }
  column "tooltip" {
    null = true
    type = text
  }
  column "status_reason" {
    null = true
    type = text
  }
  column "schedule" {
    null = true
    type = text
  }
  column "icon" {
    null = true
    type = text
  }
  column "type" {
    null = true
    type = text
  }
  column "owner" {
    null = true
    type = text
  }
  column "selectors" {
    null = true
    type = jsonb
  }
  column "log_selectors" {
    null = true
    type = jsonb
  }
  column "component_checks" {
    null = true
    type = jsonb
  }
  column "configs" {
    null = true
    type = jsonb
  }
  column "properties" {
    null = true
    type = jsonb
  }
  column "path" {
    null = true
    type = text
  }
  column "summary" {
    null = true
    type = jsonb
  }
  column "is_leaf" {
    null    = true
    type    = boolean
    default = false
  }
  column "cost_per_minute" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_1d" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_7d" {
    null = true
    type = numeric(16, 4)
  }
  column "cost_total_30d" {
    null = true
    type = numeric(16, 4)
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = true
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
  foreign_key "components_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "components_parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "components_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "components_topology_id_fkey" {
    columns     = [column.topology_id]
    ref_columns = [table.topologies.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "components_agent_id_fkey" {
    columns     = [column.agent_id]
    ref_columns = [table.agents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "idx_components_properties" {
    columns = [column.properties]
    type    = GIN
  }
  index "components_topology_id_type_name_parent_id_key" {
    unique  = true
    columns = [column.topology_id, column.type, column.name, column.parent_id]
  }
  index "components_path_is_pushed_idx" {
    on {
      expr = "length(path)"
    }
    where = "is_pushed IS FALSE"
  }
  index "idx_components_deleted_at" {
    columns = [column.deleted_at]
  }
  index "idx_components_agent" {
    columns = [column.agent_id]
  }
  index "idx_components_parent_id" {
    columns = [column.parent_id]
  }
  index "idx_components_config_id" {
    columns = [column.config_id]
  }
  index "idx_components_configs" {
    columns = [column.configs]
  }
  index "components_created_by_idx" {
    columns = [column.created_by]
  }
}

table "check_component_relationships" {
  schema = schema.public
  column "component_id" {
    null = false
    type = uuid
  }
  column "check_id" {
    null = false
    type = uuid
  }
  column "canary_id" {
    null = false
    type = uuid
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
  column "selector_id" {
    null = true
    type = text
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  foreign_key "check_component_relationships_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "check_component_relationships_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "check_component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "check_component_relationships_component_id_check_id_canary__key" {
    unique  = true
    columns = [column.component_id, column.check_id, column.canary_id, column.selector_id]
  }
  index "check_component_relationships_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
  index "idx_check_component_relationships_deleted_at" {
    columns = [column.deleted_at]
  }
  index "check_component_relationships_check_id_idx" {
    columns = [column.check_id]
  }
  index "check_component_relationships_canary_id_idx" {
    columns = [column.canary_id]
  }
}

table "config_component_relationships" {
  schema = schema.public
  column "component_id" {
    null = false
    type = uuid
  }
  column "config_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = true
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  column "selector_id" {
    null = true
    type = text
  }
  column "is_pushed" {
    null    = false
    default = false
    type    = bool
  }
  foreign_key "config_component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "config_component_relationships_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "config_component_relationships_component_id_config_id_key" {
    unique  = true
    columns = [column.component_id, column.config_id]
  }
  index "idx_config_component_relationships_deleted_at" {
    columns = [column.deleted_at]
  }
  index "config_component_relationships_is_pushed_idx" {
    columns = [column.is_pushed]
    where   = "is_pushed IS FALSE"
  }
  index "config_component_relationships_config_id_idx" {
    columns = [column.config_id]
  }
}
