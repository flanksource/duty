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
  config_access.external_role_id,
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
DROP VIEW IF EXISTS config_access_summary_by_user;
DROP VIEW IF EXISTS config_access_summary_by_config;
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

-- config_access_summary_by_user
CREATE VIEW config_access_summary_by_user AS
SELECT
  config_access_summary.external_user_id as external_user_id,
  config_access_summary."user" as "user",
  config_access_summary.email as email,
  COUNT(*) as access_count,
  COUNT(DISTINCT config_access_summary."role") as distinct_roles,
  COUNT(DISTINCT config_access_summary.config_id) as distinct_configs,
  MAX(config_access_summary.last_signed_in_at) as last_signed_in_at,
  MAX(config_access_summary.created_at) as latest_grant
FROM config_access_summary
GROUP BY config_access_summary.external_user_id, config_access_summary."user", config_access_summary.email;

-- config_access_filter_options
-- Returns distinct values for all filter dropdowns in a single call.
-- Each facet excludes its own filter parameter so that selecting a value
-- in one dropdown does not remove it from its own option list (faceted search).
DROP FUNCTION IF EXISTS config_access_filter_options;

CREATE OR REPLACE FUNCTION config_access_filter_options(
  p_config_id uuid DEFAULT NULL,
  p_config_type text DEFAULT NULL,
  p_user_id uuid DEFAULT NULL,
  p_role text DEFAULT NULL,
  p_user_type text DEFAULT NULL
) RETURNS jsonb AS $$
SELECT jsonb_build_object(
  'catalogs', COALESCE((
    SELECT jsonb_agg(to_jsonb(sub))
    FROM (
      SELECT DISTINCT config_id, config_name, config_type
      FROM config_access_summary
      WHERE (p_config_type IS NULL OR config_type = p_config_type)
        AND (p_user_id IS NULL OR external_user_id = p_user_id)
        AND (p_role IS NULL OR "role" = p_role)
        AND (p_user_type IS NULL OR user_type = p_user_type)
      ORDER BY config_name
    ) sub
  ), '[]'::jsonb),

  'users', COALESCE((
    SELECT jsonb_agg(to_jsonb(sub))
    FROM (
      SELECT DISTINCT external_user_id, "user", email
      FROM config_access_summary
      WHERE (p_config_id IS NULL OR config_id = p_config_id)
        AND (p_config_type IS NULL OR config_type = p_config_type)
        AND (p_role IS NULL OR "role" = p_role)
        AND (p_user_type IS NULL OR user_type = p_user_type)
      ORDER BY "user"
    ) sub
  ), '[]'::jsonb),

  'roles', COALESCE((
    SELECT jsonb_agg(to_jsonb(sub))
    FROM (
      SELECT DISTINCT "role"
      FROM config_access_summary
      WHERE role IS NOT NULL
        AND (p_config_id IS NULL OR config_id = p_config_id)
        AND (p_config_type IS NULL OR config_type = p_config_type)
        AND (p_user_id IS NULL OR external_user_id = p_user_id)
        AND (p_user_type IS NULL OR user_type = p_user_type)
      ORDER BY "role"
    ) sub
  ), '[]'::jsonb),

  'user_types', COALESCE((
    SELECT jsonb_agg(to_jsonb(sub))
    FROM (
      SELECT DISTINCT user_type
      FROM config_access_summary
      WHERE user_type IS NOT NULL
        AND (p_config_id IS NULL OR config_id = p_config_id)
        AND (p_config_type IS NULL OR config_type = p_config_type)
        AND (p_user_id IS NULL OR external_user_id = p_user_id)
        AND (p_role IS NULL OR "role" = p_role)
      ORDER BY user_type
    ) sub
  ), '[]'::jsonb)
);
$$ LANGUAGE sql STABLE;

-- config_access_summary_by_config
CREATE VIEW config_access_summary_by_config AS
SELECT
  config_access_summary.config_id as config_id,
  config_access_summary.config_name as config_name,
  config_access_summary.config_type as config_type,
  COUNT(*) as access_count,
  COUNT(DISTINCT config_access_summary.external_user_id) as distinct_users,
  COUNT(DISTINCT config_access_summary."role") as distinct_roles,
  MAX(config_access_summary.last_signed_in_at) as last_signed_in_at,
  MAX(config_access_summary.created_at) as latest_grant
FROM config_access_summary
GROUP BY config_access_summary.config_id, config_access_summary.config_name, config_access_summary.config_type;
