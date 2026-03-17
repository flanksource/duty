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
