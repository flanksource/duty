DO $$
BEGIN
  IF NOT EXISTS (
    SELECT
    FROM
      pg_catalog.pg_roles
    WHERE
      rolname = 'api_views_owner') THEN
  -- NOTE:In postgres v14, views are run using the view owner's permission.
  -- When RLS is enabled, we want to run the view using the current user (postgres_anon for eg.)
  -- Hence, we create a new role to make the owner of all the views that make use of RLS enabled tables.
  -- The role is created using NOBYPASSRLS option so RLS is enforced.
  CREATE ROLE api_views_owner NOSUPERUSER NOBYPASSRLS;
END IF;
END
$$;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO api_views_owner;

ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;

ALTER TABLE components ENABLE ROW LEVEL SECURITY;

-- Policy config items
DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN current_setting('request.jwt.claims', TRUE) IS NULL THEN
        TRUE
      ELSE
        (tags::jsonb @> (current_setting('request.jwt.claims', TRUE)::json ->> 'tags')::jsonb OR agent_id = ANY (ARRAY (
          SELECT
            (jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'agents'))::uuid)))
      END);

DROP POLICY IF EXISTS config_items_view_owner_allow ON config_items;

-- Policy components
DROP POLICY IF EXISTS components_auth ON components;

CREATE POLICY components_auth ON components
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN current_setting('request.jwt.claims', TRUE) IS NULL THEN
        TRUE
      ELSE
        (agent_id = ANY (ARRAY (
          SELECT
            (jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'agents'))::uuid)))
      END);

DROP POLICY IF EXISTS components_view_owner_allow ON components;

-- TODO: Add more
ALTER VIEW config_detail OWNER TO api_views_owner;

ALTER VIEW config_labels OWNER TO api_views_owner;

ALTER VIEW config_names OWNER TO api_views_owner;

ALTER VIEW config_statuses OWNER TO api_views_owner;

ALTER VIEW config_summary OWNER TO api_views_owner;

ALTER MATERIALIZED VIEW config_item_summary_3d OWNER TO api_views_owner;

ALTER MATERIALIZED VIEW config_item_summary_7d OWNER TO api_views_owner;

ALTER MATERIALIZED VIEW config_item_summary_30d OWNER TO api_views_owner;

