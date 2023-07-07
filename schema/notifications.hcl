table "notifications" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "events" {
    null = false
    type = sql("text[]")
  }
  column "template" {
    null = false
    type = text
  }
  column "receivers" {
    null    = false
    type    = jsonb
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
    null    = true
    type    = timestamptz
  }
}