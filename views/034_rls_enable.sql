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

-- Policy config items
ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN (
        current_setting('request.jwt.claims', TRUE) IS NULL 
        OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL 
      )
      THEN TRUE
      ELSE (
        agent_id = ANY (ARRAY (SELECT (jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'agents'))::uuid))
        OR 
        EXISTS  (
          SELECT 1 FROM jsonb_array_elements((current_setting('request.jwt.claims', TRUE)::json ->> 'tags')::jsonb) allowed_tags
          WHERE config_items.tags::jsonb @> allowed_tags.value
        )
      )
      END
    );

-- Policy config_changes 
ALTER TABLE config_changes ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS config_changes_auth ON config_changes;

CREATE POLICY config_changes_auth ON config_changes
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN (
        current_setting('request.jwt.claims', TRUE) IS NULL 
        OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL 
      )
      THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_changes.config_id
      )
      END
    );

-- Policy config_analysis
ALTER TABLE config_analysis ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

CREATE POLICY config_analysis_auth ON config_analysis
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN (
        current_setting('request.jwt.claims', TRUE) IS NULL 
        OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL 
      )
      THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_analysis.config_id
      )
      END
    );

-- Policy components
ALTER TABLE components ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS components_auth ON components;

CREATE POLICY components_auth ON components
  FOR ALL TO postgrest_api, postgrest_anon, api_views_owner
    USING (
      CASE WHEN (
        current_setting('request.jwt.claims', TRUE) IS NULL 
        OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL 
      )
      THEN TRUE
      ELSE (
        agent_id = ANY (ARRAY (SELECT (jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'agents'))::uuid))
      )
      END
    );

ALTER VIEW analysis_by_config OWNER TO api_views_owner;
ALTER VIEW catalog_changes OWNER TO api_views_owner;
ALTER VIEW check_summary_by_config OWNER TO api_views_owner;
ALTER VIEW check_summary_for_config OWNER TO api_views_owner;
ALTER VIEW checks_by_config OWNER TO api_views_owner;
ALTER VIEW config_analysis_analyzers OWNER TO api_views_owner;
ALTER VIEW config_analysis_by_severity OWNER TO api_views_owner;
ALTER VIEW config_analysis_items OWNER TO api_views_owner;
ALTER VIEW config_changes_by_types OWNER TO api_views_owner;
ALTER VIEW config_changes_items OWNER TO api_views_owner;
ALTER VIEW config_class_summary OWNER TO api_views_owner;
ALTER VIEW config_classes OWNER TO api_views_owner;
ALTER VIEW config_detail OWNER TO api_views_owner;
ALTER VIEW config_items_aws OWNER TO api_views_owner;
ALTER VIEW config_labels OWNER TO api_views_owner;
ALTER VIEW config_names OWNER TO api_views_owner;
ALTER VIEW config_scrapers_with_status OWNER TO api_views_owner;
ALTER VIEW config_statuses OWNER TO api_views_owner;
ALTER VIEW config_summary OWNER TO api_views_owner;
ALTER VIEW config_tags OWNER TO api_views_owner;
ALTER VIEW config_types OWNER TO api_views_owner;
ALTER VIEW configs OWNER TO api_views_owner;
ALTER VIEW incidents_by_config OWNER TO api_views_owner;
ALTER VIEW pg_config OWNER TO api_views_owner;

ALTER MATERIALIZED VIEW config_item_summary_3d OWNER TO api_views_owner;
ALTER MATERIALIZED VIEW config_item_summary_7d OWNER TO api_views_owner;
ALTER MATERIALIZED VIEW config_item_summary_30d OWNER TO api_views_owner;

