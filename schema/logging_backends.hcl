enum "source" {
  schema = schema.public
  values = ["KubernetesCRD", "ConfigFile"]
}

table "logging_backends" {
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
    type = enum.source
  }
  column "created_at" {
    null = true
    type = timestamptz
  }
  column "created_by" {
    null = true
    type = uuid
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
}
