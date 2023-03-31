
table "templates" {
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
  column "namespace" {
    null = false
    type = text
  }
  column "labels" {
    null = true
    type = jsonb
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "created_at" {
    null = true
    type = timestamp
  }
  column "updated_at" {
    null = true
    type = timestamp
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
    type = timestamp
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "templates_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "templates_name_namespace_key" {
    unique  = true
    columns = [column.name, column.namespace]
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
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamp
  }
  column "selector_id" {
    null = true
    type = text
  }
  column "relationship_path" {
    null = true
    type = text
  }
  foreign_key "component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "component_relationships_relationship_id_fkey" {
    columns     = [column.relationship_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "component_relationships_component_id_relationship_id_select_key" {
    unique  = true
    columns = [column.component_id, column.relationship_id, column.selector_id]
  }
}
table "components" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "system_template_id" {
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
  column "status" {
    null = false
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
    type = numeric(16,4)
  }
  column "cost_total_1d" {
    null = true
    type = numeric(16,4)
  }
  column "cost_total_7d" {
    null = true
    type = numeric(16,4)
  }
  column "cost_total_30d" {
    null = true
    type = numeric(16,4)
  }
  column "created_by" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamp
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
  foreign_key "components_system_template_id_fkey" {
    columns     = [column.system_template_id]
    ref_columns = [table.templates.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "components_system_template_id_type_name_parent_id_key" {
    unique  = true
    columns = [column.system_template_id, column.type, column.name, column.parent_id]
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
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamp
  }
  column "selector_id" {
    null = true
    type = text
  }
  foreign_key "check_component_relationships_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "check_component_relationships_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "check_component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "check_component_relationships_component_id_check_id_canary__key" {
    unique  = true
    columns = [column.component_id, column.check_id, column.canary_id, column.selector_id]
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
    type    = timestamp
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamp
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamp
  }
  column "selector_id" {
    null = true
    type = text
  }
  foreign_key "config_component_relationships_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "config_component_relationships_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "config_component_relationships_component_id_config_id_key" {
    unique  = true
    columns = [column.component_id, column.config_id]
  }
}
