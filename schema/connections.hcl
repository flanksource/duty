table "connections" {
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
    default = "default"
  }
  column "type" {
    null    = false
    type    = text
    comment = "The type of connection e.g. postgres, mysql, aws, gcp, http etc."
  }
  column "url" {
    null    = true
    type    = text
    comment = "A raw value or encrypted value prefixed with 'enc:' or a configmap key in the format 'config:namespace/name/key' or secret in the format 'secret:namespace/name/key', the url can include $(password) and $(username) which will be replaced with the password and username respectively."
  }
  column "source" {
    null    = true
    type    = enum.source
    comment = "Where the connection was created from."
  }
  column "username" {
    null    = true
    type    = text
    comment = "A raw value or encrypted value prefixed with 'enc:' or a configmap key in the format 'config:namespace/name/key' or secret in the format 'secret:namespace/name/key'"
  }
  column "password" {
    null    = true
    type    = text
    comment = "A raw value or encrypted value prefixed with 'enc:' or a configmap key in the format 'config:namespace/name/key' or secret in the format 'secret:namespace/name/key'"
  }
  column "certificate" {
    null = true
    type = text
  }
  column "properties" {
    null    = true
    type    = jsonb
    comment = "Used for storing connection properties e.g. region and STS endpoint for AWS connections."
  }
  column "insecure_tls" {
    null    = true
    type    = boolean
    default = true
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
  index "connections_name_namespace_key" {
    unique  = true
    columns = [column.name, column.namespace]
    where   = "deleted_at IS NULL"
  }
  index "connections_created_by_idx" {
    columns = [column.created_by]
  }
  foreign_key "connections_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
