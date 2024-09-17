table "notifications" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "name" {
    null    = false
    type    = text
    default = sql("generate_ulid()") # temporary default value to make the migration possible. we can remove this later.
  }
  column "namespace" {
    null = true
    type = text
  }
  column "events" {
    null    = false
    type    = sql("text[]")
    comment = "a list of events this notification is for."
  }
  column "error" {
    null = true
    type = text
  }
  column "title" {
    null = true
    type = text
  }
  column "template" {
    null = true
    type = text
  }
  column "filter" {
    null = true
    type = text
  }
  column "properties" {
    null    = true
    type    = jsonb
    comment = "Shoutrrr properties used when sending email to the person receipient."
  }
  column "person_id" {
    null    = true
    type    = uuid
    comment = "person who should receive this notification."
  }
  column "team_id" {
    null    = true
    type    = uuid
    comment = "team that should receive this notification."
  }
  column "repeat_interval" {
    null = true
    type = text
  }
  column "wait_for" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "group_by" {
    null    = true
    type    = sql("text[]")
    comment = "group by fields for repeat interval"
  }
  column "custom_services" {
    null    = true
    type    = jsonb
    comment = "other 3rd party services for the notification like Slack, Telegram, ..."
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
  column "source" {
    null    = true
    type    = enum.source
    comment = "Where the notification was created from."
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
  foreign_key "notification_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "notification_person_id_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  foreign_key "notification_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = CASCADE
  }
  index "notifications_name_namespace_key" {
    unique  = true
    columns = [column.name, column.namespace]
    where   = "deleted_at IS NULL"
  }
}

table "notification_send_history" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "notification_id" {
    null = false
    type = uuid
  }
  column "body" {
    null = true # nullable for unsent notifications
    type = text
  }
  column "status" {
    null = true
    type = text
  }
  column "count" {
    null    = false
    default = 1
    type    = integer
  }
  column "first_observed" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "source_event" {
    null    = false
    type    = text
    comment = "The event that caused this notification"
  }
  column "resource_id" {
    null    = false
    type    = uuid
    comment = "The resource this notification is for"
  }
  column "person_id" {
    null = true
    type = uuid
  }
  column "error" {
    null = true
    type = text
  }
  column "duration_millis" {
    null = true
    type = integer
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "notification_id_fkey" {
    columns     = [column.notification_id]
    ref_columns = [table.notifications.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_recipient_person_id_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
}

table "notification_silences" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "namespace" {
    null = false
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "from" {
    null = false
    type = timestamptz
  }
  column "until" {
    null = false
    type = timestamptz
  }
  column "recursive" {
    null = true
    type = bool
  }
  column "config_id" {
    null = true
    type = uuid
  }
  column "check_id" {
    null = true
    type = uuid
  }
  column "canary_id" {
    null = true
    type = uuid
  }
  column "component_id" {
    null = true
    type = uuid
  }
  column "source" {
    null = true
    type = enum.source
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
  index "notification_silences_from_idx" {
    type    = BRIN
    columns = [column.from]
  }
  index "notification_silences_until_idx" {
    type    = BRIN
    columns = [column.until]
  }
}