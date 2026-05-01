table "invites" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "email" {
    null = false
    type = text
  }
  column "role" {
    null = false
    type = text
  }
  column "invited_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "invited_by" {
    null = true
    type = uuid
  }
  column "accepted_at" {
    null    = true
    type    = timestamptz
    comment = "Used mainly to mark an invite as completed"
  }
  primary_key {
    columns = [column.id]
  }
  index "invites_pending_email_unique_idx" {
    unique = true
    where  = "accepted_at IS NULL"
    on {
      expr = "lower(email)"
    }
  }
  index "invites_email_idx" {
    on {
      expr = "lower(email)"
    }
  }
  index "invites_invited_by_idx" {
    columns = [column.invited_by]
  }
  foreign_key "invites_invited_by_fkey" {
    columns     = [column.invited_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}
