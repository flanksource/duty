DROP VIEW IF EXISTS configs CASCADE;

-- config_summary & config_class_summary aggregate cost straight off config_items,
-- so they hold a dependency on those columns and must go before the schema apply.
DROP VIEW IF EXISTS config_summary CASCADE;

DROP VIEW IF EXISTS config_class_summary CASCADE;

DROP VIEW IF EXISTS config_detail CASCADE;

DROP VIEW IF EXISTS config_tags CASCADE;

DROP VIEW IF EXISTS config_detail CASCADE;

DROP VIEW IF EXISTS notification_send_history_resource_tags;

DROP VIEW IF EXISTS notification_send_history_resource_types;

DROP VIEW IF EXISTS notification_send_history_summary;

DROP VIEW IF EXISTS user_config_access_summary;
