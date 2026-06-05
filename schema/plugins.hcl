table "plugins" {
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
    null    = false
    type    = text
    default = "default"
  }
  column "source" {
    null    = true
    type    = enum.source
    comment = "Where the plugin row was created from (CRD, UI, etc.)."
  }
  column "spec" {
    null    = false
    type    = jsonb
    comment = "Serialised v1.PluginSpec: binary source, version, checksum, selector, connections, sqlConnection, properties."
  }
  column "installed_path" {
    null    = true
    type    = text
    comment = "On-disk path where deps placed the binary; mirrors PluginStatus.InstalledPath."
  }
  column "plugin_version" {
    null    = true
    type    = text
    comment = "Version reported by the plugin in its manifest; mirrors PluginStatus.PluginVersion."
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
  index "plugins_name_namespace_key" {
    unique  = true
    columns = [column.name, column.namespace]
    where   = "deleted_at IS NULL"
  }
}
