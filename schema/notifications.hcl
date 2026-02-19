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
  column "error_at" {
    null = true
    type = timestamptz
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
  column "playbook_id" {
    null    = true
    type    = uuid
    comment = "playbook to trigger"
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
  column "inhibitions" {
    null = true
    type = jsonb
  }
  column "wait_for" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "wait_for_eval_period" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "group_by" {
    null    = true
    type    = sql("text[]")
    comment = "group by fields for repeat interval"
  }
  column "group_by_interval" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "watchdog_interval" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "custom_services" {
    null    = true
    type    = jsonb
    comment = "other 3rd party services for the notification like Slack, Telegram, ..."
  }
  column "fallback_delay" {
    null    = true
    type    = bigint
    comment = "duration in nanoseconds"
  }
  column "fallback_custom_services" {
    null = true
    type = jsonb
  }
  column "fallback_playbook_id" {
    null = true
    type = uuid
  }
  column "fallback_person_id" {
    null = true
    type = uuid
  }
  column "fallback_team_id" {
    null = true
    type = uuid
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
  index "notifications_created_by_idx" {
    columns = [column.created_by]
  }
  index "notifications_person_id_idx" {
    columns = [column.person_id]
  }
  index "notifications_team_id_idx" {
    columns = [column.team_id]
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
    null    = true # nullable for unsent notifications
    type    = text
    comment = "Rendered body for raw/custom templates and existing clients. Use body_payload for clicky-formatted notifications."
  }
  column "body_payload" {
    null    = true
    type    = jsonb
    comment = "schema and data payload for clicky formatting"
  }
  column "status" {
    null = true
    type = text
  }
  column "not_before" {
    null = true
    type = timestamptz
  }
  column "retries" {
    null    = true
    type    = integer
    comment = "number of retries of pending notifications"
  }
  column "payload" {
    null    = true
    type    = jsonb
    comment = "holds in original event properties for delayed/pending notifications"
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
  column "resource_health" {
    null    = true
    type    = text
    comment = "Health of the resource at the time of event"
  }
  column "resource_status" {
    null    = true
    type    = text
    comment = "Status of the resource at the time of event"
  }
  column "resource_health_description" {
    null = true
    type = text
    comment = "Health description of the resource at the time of event"
  }
  column "person_id" {
    null    = true
    type    = uuid
    comment = "recipient person"
  }
  column "team_id" {
    null    = true
    type    = uuid
    comment = "recipient team"
  }
  column "connection_id" {
    null    = true
    type    = uuid
    comment = "recipient connection"
  }
  column "silenced_by" {
    null    = true
    type    = uuid
    comment = "the notification silence that silenced this notification"
  }
  column "playbook_run_id" {
    null    = true
    type    = uuid
    comment = "playbook run created by this notification dispatch"
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
  column "group_id" {
    type    = uuid
    null    = true
    comment = "Represents the group this notification was sent for"
  }
  column "parent_id" {
    null    = true
    default = null
    type    = uuid
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [table.notification_send_history.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
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
  index "notification_send_history_notification_id_idx" {
    columns = [column.notification_id]
  }
  index "notification_send_history_parent_id_idx" {
    columns = [column.parent_id]
  }
  index "notification_send_history_person_id_idx" {
    columns = [column.person_id]
  }
  index "notification_send_history_resource_id_idx" {
    columns = [column.resource_id]
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
    null = true
    type = text
  }
  column "name" {
    null    = false
    type    = text
    default = sql("generate_ulid()") # temporary default value to make the migration possible. we can remove this later.
  }
  column "description" {
    null = true
    type = text
  }
  column "filter" {
    null = true
    type = text
  }
  column "selectors" {
    null    = true
    type    = jsonb
    comment = "list of resource selectors"
  }
  column "error" {
    null = true
    type = text
  }
  column "from" {
    null = true
    type = timestamptz
  }
  column "until" {
    null = true
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
  foreign_key "notification_silence_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_silence_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_silence_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_silence_canary_id_fkey" {
    columns     = [column.canary_id]
    ref_columns = [table.canaries.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_silence_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "notification_silences_check_id_idx" {
    columns = [column.check_id]
  }
  index "notification_silences_config_id_idx" {
    columns = [column.config_id]
  }
  index "notification_silences_component_id_idx" {
    columns = [column.component_id]
  }
  index "notification_silences_canary_id_idx" {
    columns = [column.canary_id]
  }
  index "notification_silences_created_by_idx" {
    columns = [column.created_by]
  }
}

table "notification_groups" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "hash" {
    null = false
    type = text
  }
  column "notification_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "notification_groups_notification_id_fkey" {
    columns     = [column.notification_id]
    ref_columns = [table.notifications.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "notification_groups_notification_id_idx" {
    columns = [column.notification_id]
  }
  index "notification_groups_hash_notification_id_idx" {
    columns = [column.hash, column.notification_id]
  }
}

table "notification_group_resources" {
  schema = schema.public
  column "group_id" {
    null = false
    type = uuid
  }
  column "config_id" {
    null = true
    type = uuid
  }
  column "check_id" {
    null = true
    type = uuid
  }
  column "component_id" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null = true
    type = timestamptz
  }
  column "resolved_at" {
    null    = true
    type    = timestamptz
    comment = "The resource was resolved and removed from the group"
  }
  foreign_key "notification_group_resources_group_id_fkey" {
    columns     = [column.group_id]
    ref_columns = [table.notification_groups.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_group_resources_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_group_resources_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  foreign_key "notification_group_resources_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = CASCADE
    on_delete   = CASCADE
  }
  index "notification_group_resources_group_id_idx" {
    columns = [column.group_id]
  }
  index "notification_group_resources_config_id_idx" {
    columns = [column.config_id]
  }
  index "notification_group_resources_check_id_idx" {
    columns = [column.check_id]
  }
  index "notification_group_resources_component_id_idx" {
    columns = [column.component_id]
  }
}
