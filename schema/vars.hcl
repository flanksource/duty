variable "uuid_nil" {
    type = string
    default = "00000000-0000-0000-0000-000000000000"
}

enum "source" {
  schema = schema.public
  # The "Topology" value cannot be dropped as migration drops the enum
  # and tries to recreate it
  # TODO: Create new enum, replace usage and delete this one
  values = ["KubernetesCRD", "ConfigFile", "UI", "Topology", "Push", "CRDSync", "ApplicationCRD", "System"]
}
