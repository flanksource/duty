table "scope_targets" {
  schema  = schema.public
  comment = "The targets of every valid Scope, one row of values each. A condition a target doesn't set is NULL and matches anything. Whole-type targets aren't stored here (see scope_members)."

  column "scope_id" {
    null = false
    type = uuid
  }

  column "resource_type" {
    null = false
    type = text
  }

  column "id" {
    null = true
    type = uuid
  }

  column "name" {
    null    = true
    type    = text
    comment = "exact name"
  }

  column "name_prefix" {
    null    = true
    type    = text
    comment = "names that start with it"
  }

  column "namespace" {
    null = true
    type = text
  }

  column "agent_id" {
    null    = true
    type    = uuid
    comment = "the agent, resolved to its id when the Scope was saved"
  }

  column "types" {
    null    = true
    type    = sql("text[]")
    comment = "any of them"
  }

  column "tags" {
    null    = true
    type    = jsonb
    comment = "key=value pairs the resource's tags must contain"
  }

  column "labels" {
    null    = true
    type    = jsonb
    comment = "key=value pairs the resource's labels must contain"
  }

  column "match_keys" {
    null    = false
    type    = sql("text[]")
    default = sql("'{}'::text[]")
    comment = "the tag, label and namespace conditions as keys (see scope_match_keys), set by trigger. A written resource is only matched against targets sharing one of its keys, and targets with none"
  }

  index "scope_targets_scope_id_idx" {
    columns = [column.scope_id]
  }

  index "scope_targets_match_keys_idx" {
    type    = GIN
    columns = [column.match_keys]
  }

  index "scope_targets_unkeyed_idx" {
    columns = [column.resource_type]
    where   = "match_keys = '{}'::text[]"
  }
}

table "scope_members" {
  schema  = schema.public
  comment = "The resources each valid Scope selects. A row with no resource_id selects every resource of the type."

  column "scope_id" {
    null = false
    type = uuid
  }

  column "resource_type" {
    null = false
    type = text
  }

  column "resource_id" {
    null = true
    type = uuid
  }

  index "scope_members_resource_key" {
    unique  = true
    columns = [column.scope_id, column.resource_type, column.resource_id]
    where   = "resource_id IS NOT NULL"
  }

  index "scope_members_whole_type_key" {
    unique  = true
    columns = [column.scope_id, column.resource_type]
    where   = "resource_id IS NULL"
  }

  index "scope_members_resource_idx" {
    columns = [column.resource_type, column.resource_id]
  }
}
