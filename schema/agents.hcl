table "agents" {
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

  column "hostname" {
    null = true
    type = text
  }

  column "description" {
    null = true
    type = text
  }

  column "ip" {
    null = true
    type = text
  }

  column "version" {
    null = true
    type = text
  }

  column "username" {
    null = true
    type = text
  }

  column "person_id" {
    null = true
    type = uuid
  }

  column "properties" {
    null = true
    type = jsonb
  }

  column "tls" {
    null = true
    type = text
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

  column "cleanup" {
    null    = false
    type    = boolean
    default = false
  }

  column "deleted_at" {
    null = true
    type = timestamptz
  }

  column "last_seen" {
    null = true
    type = timestamptz
  }

  column "last_received" {
    null = true
    type = timestamptz
  }

  primary_key {
    columns = [column.id]
  }

  index "agents_name_key" {
    unique  = true
    columns = [column.name]
    where   = "deleted_at IS NULL"
  }
  index "idx_agents_name" {
    columns = [column.name]
  }

  foreign_key "agents_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
