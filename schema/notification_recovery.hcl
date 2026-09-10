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
  column "wake_pending" {
    type = boolean
    null = false
    default = true
  }
  column "deletion_observed_at" {
    type = timestamptz
    null = true
  }
  primary_key {
    columns = [column.resource_type, column.resource_id]
  }
  index "notification_health_states_wake_idx" {
    columns = [column.resource_type, column.resource_id]
    where = "((health = 'healthy'::text) AND wake_pending)"
  }
  index "notification_health_states_episode_idx" {
    columns = [column.episode_id]
  }
  index "notification_health_states_deleted_idx" {
    columns = [column.deletion_observed_at]
    where = "(deletion_observed_at IS NOT NULL)"
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
  index "notification_health_episodes_resource_idx" {
    columns = [column.resource_type, column.resource_id, column.id]
  }
  index "notification_health_episodes_retention_idx" {
    columns = [column.healthy_at]
    where = "(healthy_at IS NOT NULL)"
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
  column "exhausted_at" {
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
  index "notification_deliveries_notification_id_idx" {
    columns = [column.notification_id]
  }
  index "notification_deliveries_episode_id_idx" {
    columns = [column.episode_id]
  }
  index "notification_deliveries_pending_idx" {
    columns = [column.not_before]
    where = "((resolved_at IS NULL) AND (sent_at IS NOT NULL) AND (status <> 'recovery-exhausted'::text) AND (status <> 'waiting-for-healthy'::text))"
  }
  index "notification_deliveries_wake_idx" {
    columns = [column.episode_id, column.id]
    where = "((resolved_at IS NULL) AND (sent_at IS NOT NULL) AND (status = ANY (ARRAY['sent'::text, 'waiting-for-healthy'::text])))"
  }
  index "notification_deliveries_resolved_retention_idx" {
    columns = [column.resolved_at]
    where = "((resolved_at IS NOT NULL) AND (sent_at IS NOT NULL) AND (status = 'resolved'::text))"
  }
  index "notification_deliveries_exhausted_retention_idx" {
    columns = [column.exhausted_at]
    where = "((exhausted_at IS NOT NULL) AND (sent_at IS NOT NULL) AND (status = 'recovery-exhausted'::text))"
  }
}
