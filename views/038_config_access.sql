-- dependsOn: functions/drop.sql

-- stale view
DROP VIEW IF EXISTS user_config_access_summary;

-- config_access_unwrapped
-- flattens config access permissions by expanding group memberships
CREATE OR REPLACE VIEW config_access_unwrapped AS
SELECT
  generate_ulid()::TEXT as id,
  config_access.config_id,
  external_user_groups.external_user_id,
  config_access.external_group_id AS external_group_id,
  NULL AS external_role_id,
  config_access.created_at,
  config_access.deleted_at,
  config_access.deleted_by,
  config_access.last_reviewed_at,
  config_access.last_reviewed_by,
  config_access.created_by,
  config_access.scraper_id
FROM
  config_access
  INNER JOIN external_user_groups ON config_access.external_group_id = external_user_groups.external_group_id
  AND config_access.deleted_at IS NULL
  AND config_access.external_group_id IS NOT NULL
UNION ALL
SELECT
  id,
  config_id,
  external_user_id,
  NULL AS external_group_id,
  external_role_id,
  created_at,
  deleted_at,
  deleted_by,
  last_reviewed_at,
  last_reviewed_by,
  created_by,
  scraper_id
  FROM config_access
  WHERE external_group_id IS NULL;

-- config_access_summary
DROP VIEW IF EXISTS config_access_summary;

CREATE VIEW config_access_summary AS
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
  config_access_logs.created_at as last_signed_in_at,
  config_access_unwrapped.last_reviewed_at as last_reviewed_at,
  config_access_unwrapped.last_reviewed_by as last_reviewed_by
FROM config_access_unwrapped
JOIN config_items ON config_access_unwrapped.config_id = config_items.id
JOIN external_users ON config_access_unwrapped.external_user_id = external_users.id
LEFT JOIN external_roles ON config_access_unwrapped.external_role_id = external_roles.id
LEFT JOIN config_access_logs
  ON config_access_unwrapped.config_id = config_access_logs.config_id AND
  config_access_unwrapped.external_user_id = config_access_logs.external_user_id
WHERE config_access_unwrapped.deleted_at IS NULL;
