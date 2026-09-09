table "notification_health_states" {
  schema = schema.public
  column "resource_type" {
    type = text
    null = false
  }
  column "resource_id" {
    type = uuid
    null = false
  }
  column "generation" {
    type = uuid
    null = false
  }
  column "episode_id" {
    type = uuid
    null = true
  }
  column "health" {
    type = text
    null = false
  }
  column "healthy_since" {
    type = timestamptz
    null = true
  }
  primary_key {
    columns = [column.resource_type, column.resource_id]
  }
}

table "notification_health_episodes" {
  schema = schema.public
  column "id" {
    type = uuid
    null = false
  }
  column "resource_type" {
    type = text
    null = false
  }
  column "resource_id" {
    type = uuid
    null = false
  }
  column "started_at" {
    type = timestamptz
    null = false
    default = sql("now()")
  }
  column "healthy_at" {
    type = timestamptz
    null = true
  }
  primary_key {
    columns = [column.id]
  }
}

table "notification_deliveries" {
  schema = schema.public
  column "id" {
    type = uuid
    null = false
  }
  column "notification_id" {
    type = uuid
    null = false
  }
  column "episode_id" {
    type = uuid
    null = false
  }
  column "history_id" {
    type = uuid
    null = true
  }
  column "connection_id" {
    type = uuid
    null = true
  }
  column "transport" {
    type = text
    null = false
  }
  column "destination" {
    type = jsonb
    null = false
  }
  column "policy" {
    type = jsonb
    null = false
  }
  column "message_id" {
    type = text
    null = false
  }
  column "channel" {
    type = text
    null = true
  }
  column "status" {
    type = text
    null = false
    default = "prepared"
  }
  column "sent_at" {
    type = timestamptz
    null = true
  }
  column "reply_at" {
    type = timestamptz
    null = true
  }
  column "reaction_at" {
    type = timestamptz
    null = true
  }
  column "resolved_at" {
    type = timestamptz
    null = true
  }
  column "not_before" {
    type = timestamptz
    null = false
    default = sql("now()")
  }
  column "lease_until" {
    type = timestamptz
    null = true
  }
  column "lease_token" {
    type = uuid
    null = true
  }
  column "attempts" {
    type = integer
    null = false
    default = 0
  }
  column "error" {
    type = text
    null = true
  }
  column "created_at" {
    type = timestamptz
    null = false
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "notification_deliveries_notification_id_fkey" {
    columns = [column.notification_id]
    ref_columns = [table.notifications.column.id]
    on_delete = CASCADE
  }
  foreign_key "notification_deliveries_episode_id_fkey" {
    columns = [column.episode_id]
    ref_columns = [table.notification_health_episodes.column.id]
    on_delete = NO_ACTION
  }
  foreign_key "notification_deliveries_history_id_fkey" {
    columns = [column.history_id]
    ref_columns = [table.notification_send_history.column.id]
    on_delete = SET_NULL
  }
  index "notification_deliveries_history_id_key" {
    unique = true
    columns = [column.history_id]
  }
  index "notification_deliveries_pending_idx" {
    columns = [column.not_before]
    where = "resolved_at IS NULL"
  }
}

