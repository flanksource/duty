table "short_urls" {
  schema = schema.public
  column "alias" {
    null = false
    type = text
    comment = "Short alias for the URL - primary key"
  }
  column "url" {
    null = false
    type = text
    comment = "Full URL to redirect to"
  }
  column "expires_at" {
    null = true
    type = timestamptz
    comment = "Optional expiry date"
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.alias]
  }
}