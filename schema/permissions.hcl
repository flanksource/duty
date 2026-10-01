table "permissions" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }

  column "name" {
    null    = false
    type    = text
    default = sql("generate_ulid()") # temporary to support migration. can be removed later.
  }

  column "namespace" {
    null = true
    type = text
  }

  column "description" {
    null = true
    type = text
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
  }

  column "subject_type" {
    null    = false
    type    = text
    default = "group"
  }

  column "subject" {
    # This should be non-nullable.
    # But since we're only adding this column now, we will use the migration to populate the value for existing records
    # and then make it non nullable.
    null = true
    type = text
  }

  column "action" {
    null = false
    type = text
  }

  column "object" {
    null = true
    type = text
  }

  column "object_selector" {
    null    = true
    type    = jsonb
    comment = "list of resource selectors to select the viable object"
  }

  column "deny" {
    type    = boolean
    default = false
  }

  column "component_id" {
    null    = true
    type    = uuid
    comment = "component resource"
  }

  column "config_id" {
    null    = true
    type    = uuid
    comment = "config item resource"
  }

  column "canary_id" {
    null    = true
    type    = uuid
    comment = "canary resource"
  }

  column "playbook_id" {
    null    = true
    type    = uuid
    comment = "playbook resource"
  }

  column "connection_id" {
    null    = true
    type    = uuid
    comment = "connection resource"
  }

  column "created_by" {
    null = true
    type = uuid
  }

  # Deprecated. Use subject and subject_type instead.
  column "person_id" {
    null = true
    type = uuid
  }

  # Deprecated. Use subject and subject_type instead.
  column "team_id" {
    null = true
    type = uuid
  }

  # Deprecated. Use object and object_selector instead.
  column "notification_id" {
    null = true
    type = uuid
  }

  column "updated_by" {
    null = true
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

  column "error" {
    null    = true
    type    = text
    comment = "stores error when invalid object selector is provided. eg: granting access to a non-existent scope"
  }

  column "until" {
    null = true
    type = timestamptz
  }

  column "agents" {
    null    = true
    type    = sql("text[]")
    comment = "a list of agent ids a user is allowed to access when row-level security is enabled"
  }

  column "tags" {
    null    = true
    type    = jsonb
    comment = "a list of tags user is allowed to access when row-level security is enabled"
  }

  primary_key {
    columns = [column.id]
  }

  check "permissions_selector_or_id_check" {
    expr = "(object_selector IS NOT NULL)::int + (NULLIF(object, '') IS NOT NULL)::int + (config_id IS NOT NULL)::int + (playbook_id IS NOT NULL)::int + (canary_id IS NOT NULL)::int + (component_id IS NOT NULL)::int + (connection_id IS NOT NULL)::int = 1"
  }

  foreign_key "permissions_playbook_id_fkey" {
    columns     = [column.playbook_id]
    ref_columns = [table.playbooks.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_connection_id_fkey" {
    columns     = [column.connection_id]
    ref_columns = [table.connections.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "permissions_notification_fkey" {
    columns     = [column.notification_id]
    ref_columns = [table.notifications.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_person_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "permissions_team_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }

  index "permissions_config_id_idx" {
    columns = [column.config_id]
  }
  index "permissions_component_id_idx" {
    columns = [column.component_id]
  }
  index "permissions_canary_id_idx" {
    columns = [column.canary_id]
  }
  index "permissions_playbook_id_idx" {
    columns = [column.playbook_id]
  }
  index "permissions_connection_id_idx" {
    columns = [column.connection_id]
  }
  index "permissions_person_id_idx" {
    columns = [column.person_id]
  }
  index "permissions_team_id_idx" {
    columns = [column.team_id]
  }
  index "permissions_notification_id_idx" {
    columns = [column.notification_id]
  }
  index "permissions_created_by_idx" {
    columns = [column.created_by]
  }
}

table "permission_groups" {
  schema = schema.public

  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }

  column "name" {
    null = true
    type = text
  }

  column "namespace" {
    null = true
    type = text
  }

  column "selectors" {
    null    = false
    type    = jsonb
    comment = "a list of selectors (a toned down version of resource selector) to select primary mission control resources like - notifications, scrapers, playbooks, ..."
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
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

  index "permission_namespace_name_key" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  index "permission_groups_created_by_idx" {
    columns = [column.created_by]
  }

  foreign_key "permissions_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "roles" {
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

  column "description" {
    null = true
    type = text
  }

  column "rules" {
    null    = false
    type    = jsonb
    default = "[]"
    comment = "rules as declared on the role: each allows or denies an action on the resources selected by a scope, optionally on the targets selected by another scope. They are compiled into casbin policies filed under binding:<namespace>/<name> for each binding of the role"
  }

  column "error" {
    null    = true
    type    = text
    comment = "why the role isn't in effect, e.g. a rule references a scope that doesn't exist. NULL when it's valid"
  }

  column "error_reason" {
    null    = true
    type    = text
    comment = "machine readable reason for error, e.g. ScopeNotFound or RowLevelSecurityRequired. NULL when it's valid"
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
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

  index "roles_namespace_name_key" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  index "roles_created_by_idx" {
    columns = [column.created_by]
  }

  foreign_key "roles_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "role_bindings" {
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

  column "description" {
    null = true
    type = text
  }

  column "role" {
    null    = false
    type    = text
    comment = "name of the bound role, in the same namespace as the binding"
  }

  column "constraints" {
    null    = false
    type    = jsonb
    default = "[]"
    comment = "constraints that narrow rules of the role, each naming a rule and the scopes, in the same namespace as the binding, its resource and target must also belong to. When there are constraints, only the allow rules they name are granted"
  }

  column "subjects" {
    null    = false
    type    = jsonb
    default = "{}"
    comment = "selectors for the people, teams, roles, OIDC identities and resources the role is bound to"
  }

  column "error" {
    null    = true
    type    = text
    comment = "why the role binding isn't in effect, e.g. a constraint names a rule the role doesn't have. NULL when it's valid"
  }

  column "error_reason" {
    null    = true
    type    = text
    comment = "machine readable reason for error, e.g. RoleNotFound or RowLevelSecurityRequired. NULL when it's valid"
  }

  column "source" {
    null    = false
    type    = text
    default = "UI"
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

  index "role_bindings_namespace_name_key" {
    unique  = true
    columns = [column.namespace, column.name]
    where   = "deleted_at IS NULL"
  }

  index "role_bindings_created_by_idx" {
    columns = [column.created_by]
  }

  foreign_key "role_bindings_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
