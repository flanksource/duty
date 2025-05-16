-- dependsOn: functions/drop.sql

CREATE OR REPLACE VIEW user_config_access_summary AS
SELECT 
  config_items.id as config_id,
  config_items.name as config_name,
  config_items.type as config_type,
  external_users.name as external_user_name,
  external_users.email as external_user_email,
  config_access.created_at as created_at,
  config_access.deleted_at as deleted_at,
  config_access.created_by as created_by,
  config_access.last_reviewed_at as last_reviewed_at,
  config_access.last_reviewed_by as last_reviewed_by
 FROM config_access
JOIN config_items ON config_access.config_id = config_items.id
JOIN external_users ON config_access.external_user_id = external_users.id
WHERE config_access.deleted_at IS NULL;