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

  # match_keys lets us find the Scopes a new resource belongs to with an index lookup,
  # instead of running every Scope's matcher against it.
  #
  # Each target stores the exact values it asks for, e.g. `namespace: team-42` is stored
  # as {ns:team-42}. A new pod in namespace team-42 looks up its own values in the index
  # on this column, and gets back only the targets asking for them. Only those are run
  # through the matcher.
  #
  # Example: there are 1,000 targets, and one of them is `namespace: payments`. A new pod
  # in namespace payments looks up {ns:payments} and gets back just that one target, so
  # the matcher runs once instead of 1,000 times.
  #
  # Targets with no exact values, e.g. `name: prod-*`, have an empty list and are always
  # run through the matcher. Filled in by a trigger; don't write it yourself.
  column "match_keys" {
    null    = false
    type    = sql("text[]")
    default = sql("'{}'::text[]")
    comment = "The exact values this target asks for, e.g. {ns:team-42,tag:env=prod}. A new resource looks up its own values here, through an index, to find the few targets worth matching instead of matching all of them. Filled in by a trigger."
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
