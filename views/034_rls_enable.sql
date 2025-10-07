CREATE
OR REPLACE FUNCTION is_rls_disabled () RETURNS BOOLEAN AS $$
BEGIN
  RETURN (current_setting('request.jwt.claims', TRUE) IS NULL
    OR current_setting('request.jwt.claims', TRUE) = ''
    OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Generic function to match a row against an array of scopes
-- Returns TRUE if the row matches ANY scope in the array (OR logic between scopes)
-- Within a scope, ALL non-empty fields must match (AND logic within scope)
CREATE OR REPLACE FUNCTION match_scope(
    scopes jsonb,           -- Array of scope objects from JWT claims
    row_tags jsonb,         -- The row's tags (can be NULL)
    row_agent uuid,         -- The row's agent_id (can be NULL)
    row_name text           -- The row's name (can be NULL)
) RETURNS BOOLEAN AS $$
DECLARE
    scope jsonb;
    scope_tags jsonb;
    scope_agents jsonb;
    scope_names jsonb;
    tags_match boolean;
    agents_match boolean;
    names_match boolean;
BEGIN
    -- If scopes is NULL or not an array or empty, deny access
    IF scopes IS NULL
       OR jsonb_typeof(scopes) != 'array'
       OR jsonb_array_length(scopes) = 0 THEN
        RETURN FALSE;
    END IF;

    -- Iterate through each scope (OR logic between scopes)
    FOR scope IN SELECT * FROM jsonb_array_elements(scopes)
    LOOP
        -- Extract fields from scope
        scope_tags := scope->'tags';
        scope_agents := scope->'agents';
        scope_names := scope->'names';

        -- Check tags match (row must contain all scope tags)
        IF scope_tags IS NULL OR jsonb_typeof(scope_tags) = 'null' OR scope_tags = '{}'::jsonb THEN
            tags_match := TRUE;
        ELSIF row_tags IS NULL THEN
            tags_match := FALSE;
        ELSE
            tags_match := row_tags @> scope_tags;
        END IF;

        -- Check agents match (row agent must be in list or wildcard)
        IF scope_agents IS NULL OR jsonb_typeof(scope_agents) = 'null' OR jsonb_array_length(scope_agents) = 0 THEN
            agents_match := TRUE;
        ELSIF scope_agents = '["*"]'::jsonb THEN
            agents_match := row_agent IS NOT NULL;
        ELSIF row_agent IS NULL THEN
            agents_match := FALSE;
        ELSE
            agents_match := scope_agents @> to_jsonb(row_agent::text);
        END IF;

        -- Check names match (row name must be in list or wildcard)
        IF scope_names IS NULL OR jsonb_typeof(scope_names) = 'null' OR jsonb_array_length(scope_names) = 0 THEN
            names_match := TRUE;
        ELSIF scope_names = '["*"]'::jsonb THEN
            names_match := row_name IS NOT NULL;
        ELSIF row_name IS NULL THEN
            names_match := FALSE;
        ELSE
            names_match := scope_names @> to_jsonb(row_name);
        END IF;

        -- If ALL conditions match (AND logic within scope), return TRUE
        IF tags_match AND agents_match AND names_match THEN
            RETURN TRUE;
        END IF;
    END LOOP;

    -- No scope matched
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Enable RLS for tables
DO $$
BEGIN
    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_items') THEN
        EXECUTE 'ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_changes') THEN
        EXECUTE 'ALTER TABLE config_changes ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_analysis') THEN
        EXECUTE 'ALTER TABLE config_analysis ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'components') THEN
        EXECUTE 'ALTER TABLE components ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_component_relationships') THEN
        EXECUTE 'ALTER TABLE config_component_relationships ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_relationships') THEN
        EXECUTE 'ALTER TABLE config_relationships ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'canaries') THEN
        EXECUTE 'ALTER TABLE canaries ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbooks') THEN
        EXECUTE 'ALTER TABLE playbooks ENABLE ROW LEVEL SECURITY;';
    END IF;
END $$;

-- Policy config items
DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        match_scope(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'config',
          config_items.tags,
          config_items.agent_id,
          config_items.name
        )
      END
    );

-- Policy config_changes 
DROP POLICY IF EXISTS config_changes_auth ON config_changes;

CREATE POLICY config_changes_auth ON config_changes
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_changes.config_id
      )
      END
    );

-- Policy config_analysis
DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

CREATE POLICY config_analysis_auth ON config_analysis
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_analysis.config_id
      )
      END
    );

-- Policy config_relationships
DROP POLICY IF EXISTS config_relationships_auth ON config_relationships;

CREATE POLICY config_relationships_auth ON config_relationships
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE (
        -- just leverage the RLS on config_items - user must have access to both items
        EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_relationships.config_id)
        AND EXISTS (SELECT 1 FROM config_items WHERE config_items.id = config_relationships.related_id)
      )
      END
    );

-- Policy config_component_relationships
DROP POLICY IF EXISTS config_component_relationships_auth ON config_component_relationships;

CREATE POLICY config_component_relationships_auth ON config_component_relationships
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_component_relationships.config_id
      )
      END
    );

-- Policy components
DROP POLICY IF EXISTS components_auth ON components;

CREATE POLICY components_auth ON components
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        match_scope(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'component',
          NULL,
          components.agent_id,
          components.name
        )
      END
    );

-- Policy canaries
DROP POLICY IF EXISTS canaries_auth ON canaries;

CREATE POLICY canaries_auth ON canaries
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        match_scope(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'canary',
          NULL,
          NULL,
          canaries.name
        )
      END
    );

-- Policy playbooks
DROP POLICY IF EXISTS playbooks_auth ON playbooks;

CREATE POLICY playbooks_auth ON playbooks
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        match_scope(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'playbook',
          NULL,
          NULL,
          playbooks.name
        )
      END
    );

ALTER VIEW analysis_by_config SET (security_invoker = true);
ALTER VIEW catalog_changes SET (security_invoker = true);
ALTER VIEW check_summary SET (security_invoker = true);
ALTER VIEW check_summary_by_config SET (security_invoker = true);
ALTER VIEW check_summary_for_config SET (security_invoker = true);
ALTER VIEW checks_by_config SET (security_invoker = true);
ALTER VIEW checks_labels_keys SET (security_invoker = true);
ALTER VIEW component_labels_keys SET (security_invoker = true);
ALTER VIEW config_analysis_analyzers SET (security_invoker = true);
ALTER VIEW config_analysis_by_severity SET (security_invoker = true);
ALTER VIEW config_analysis_items SET (security_invoker = true);
ALTER VIEW config_changes_by_types SET (security_invoker = true);
ALTER VIEW config_class_summary SET (security_invoker = true);
ALTER VIEW config_classes SET (security_invoker = true);
ALTER VIEW config_detail SET (security_invoker = true);
ALTER VIEW config_labels SET (security_invoker = true);
ALTER VIEW config_names SET (security_invoker = true);
ALTER VIEW config_scrapers_with_status SET (security_invoker = true);
ALTER VIEW config_statuses SET (security_invoker = true);
ALTER VIEW config_summary SET (security_invoker = true);
ALTER VIEW config_tags SET (security_invoker = true);
ALTER VIEW config_tags_labels_keys SET (security_invoker = true);
ALTER VIEW config_types SET (security_invoker = true);
ALTER VIEW configs SET (security_invoker = true);
ALTER VIEW topology SET (security_invoker = true);
ALTER VIEW incidents_by_config SET (security_invoker = true);
ALTER VIEW playbook_names SET (security_invoker = true);

-- TODO: Move 034_rls_enable.sql as the last script (eg: 10000_rls_enable.sql)
-- So that all the views are already created before it runs.