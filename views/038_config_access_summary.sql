-- dependsOn: 038_config_access.sql

-- config_access_summary
DROP VIEW IF EXISTS config_access_summary;

CREATE VIEW config_access_summary AS
WITH access_log_agg AS (
  SELECT
    config_id,
    external_user_id,
    MAX(CASE WHEN outcome = 'allowed' THEN created_at END) AS last_signed_in_at,
    MAX(created_at) AS last_access_attempt_at,
    SUM(CASE WHEN outcome = 'allowed' THEN COALESCE(count, 1) ELSE 0 END)::integer AS allowed_count,
    SUM(CASE WHEN outcome = 'denied' THEN COALESCE(count, 1) ELSE 0 END)::integer AS denied_count
  FROM config_access_logs
  GROUP BY config_id, external_user_id
)
SELECT
  config_items.id as config_id,
  config_items.name as config_name,
  config_items.type as config_type,
  config_access_unwrapped.external_group_id as external_group_id,
  config_access_unwrapped.external_user_id as external_user_id,
  external_roles.name as "role",
  external_users.name as "user",
  external_users.email as "email",
  external_users.user_type as user_type,
  config_access_unwrapped.created_at as created_at,
  config_access_unwrapped.deleted_at as deleted_at,
  config_access_unwrapped.created_by as created_by,
  ala.last_signed_in_at as last_signed_in_at,
  ala.last_access_attempt_at as last_access_attempt_at,
  ala.allowed_count as allowed_count,
  ala.denied_count as denied_count,
  config_access_unwrapped.last_reviewed_at as last_reviewed_at,
  config_access_unwrapped.last_reviewed_by as last_reviewed_by
FROM config_access_unwrapped
JOIN config_items ON config_access_unwrapped.config_id = config_items.id
JOIN external_users ON config_access_unwrapped.external_user_id = external_users.id
LEFT JOIN external_roles ON config_access_unwrapped.external_role_id = external_roles.id
LEFT JOIN access_log_agg ala
  ON config_access_unwrapped.config_id = ala.config_id
  AND config_access_unwrapped.external_user_id = ala.external_user_id
WHERE config_access_unwrapped.deleted_at IS NULL;
