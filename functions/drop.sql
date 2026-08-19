DROP VIEW IF EXISTS configs CASCADE;
DROP FUNCTION IF EXISTS refresh_config_cost_summary CASCADE;

-- The materialized view depends on config_costs columns. Drop it before Atlas changes
-- the table, then recreate it from the current definition in 006_config_views.sql.
DROP MATERIALIZED VIEW IF EXISTS config_cost_summary CASCADE;

-- Former name of config_cost_summary; dropped so upgrades do not leave it behind
-- holding a stale dependency on columns the schema apply is about to change.
DROP FUNCTION IF EXISTS refresh_config_costs_rollup CASCADE;
DROP MATERIALIZED VIEW IF EXISTS config_costs_rollup CASCADE;

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
