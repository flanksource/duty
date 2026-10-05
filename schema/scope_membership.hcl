table "scope_targets" {
  schema  = schema.public
  comment = "The selectors of every valid Scope, flattened to plain columns so SQL can match resources against them. Each row is one target of a Scope (one ResourceSelector); a resource matches the row when every non-NULL column holds for it, and a NULL column matches anything. Resource triggers and scope_rebuild_members read these rows to keep scope_members up to date. Whole-type targets (name \"*\") aren't stored here, see scope_members."

  column "scope_id" {
    null = false
    type = uuid
  }

  column "resource_type" {
    null = false
    type = text
  }

  column "resource_id" {
    null    = true
    type    = uuid
    comment = "the selector's id: the one resource the target selects"
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
    comment = "the tag, label and namespace conditions as keys, set by trigger from tags, labels and namespace (see scope_match_keys). E.g. tags {\"env\":\"prod\"}, labels {\"team\":\"x\"} and namespace \"default\" give {tag:env=prod,label:team=x,ns:default}. A written resource is only matched against targets sharing one of its keys (through the GIN index), and targets with none"
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
