-- Generic function to match a row against an array of scopes
-- Returns TRUE if the row matches ANY scope in the array (OR logic between scopes)
-- Within a scope, ALL non-empty fields must match (AND logic within scope)
CREATE OR REPLACE FUNCTION match_scope(
    scopes jsonb,           -- Array of scope objects from JWT claims
    row_tags jsonb,         -- The row's tags (can be NULL)
    row_agent uuid,         -- The row's agent_id (can be NULL)
    row_name text,          -- The row's name (can be NULL)
    row_id uuid             -- The row's ID (can be NULL)
) RETURNS BOOLEAN AS $$
DECLARE
    scope jsonb;
    scope_tags jsonb;
    scope_agents jsonb;
    scope_names jsonb;
    scope_id text;
    tags_match boolean;
    agents_match boolean;
    names_match boolean;
    id_match boolean;
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
        scope_id := NULLIF(btrim(scope->>'id'), '');

        -- Check if scope has any fields applicable to this resource type
        -- A field is applicable if: scope defines it AND resource supports it (row param not NULL)
        -- If no applicable fields, scope is effectively empty for this resource type
        IF ((scope_tags IS NULL OR scope_tags = '{}'::jsonb) OR row_tags IS NULL)
           AND (COALESCE(jsonb_array_length(scope_agents), 0) = 0 OR row_agent IS NULL)
           AND (COALESCE(jsonb_array_length(scope_names), 0) = 0 OR row_name IS NULL)
           AND (scope_id IS NULL OR row_id IS NULL) THEN
            CONTINUE;
        END IF;

        -- Check tags match (row must contain all scope tags)
        IF scope_tags IS NULL OR jsonb_typeof(scope_tags) = 'null' OR scope_tags = '{}'::jsonb THEN
            tags_match := TRUE;
        ELSIF row_tags IS NULL THEN
            tags_match := TRUE; -- Resource doesn't have tags, ignore this check
        ELSE
            tags_match := row_tags @> scope_tags;
        END IF;

        -- Check agents match (row agent must be in list or wildcard)
        IF scope_agents IS NULL OR jsonb_typeof(scope_agents) = 'null' OR jsonb_array_length(scope_agents) = 0 THEN
            agents_match := TRUE;
        ELSIF row_agent IS NULL THEN
            agents_match := TRUE; -- Resource doesn't have agents, ignore this check
        ELSIF scope_agents = '["*"]'::jsonb THEN
            agents_match := row_agent IS NOT NULL;
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

        -- Check ID match (row ID must match if provided)
        IF scope_id IS NULL THEN
            id_match := TRUE;
        ELSIF row_id IS NULL THEN
            id_match := FALSE;
        ELSIF scope_id = '*' THEN
            id_match := row_id IS NOT NULL;
        ELSE
            id_match := lower(scope_id) = row_id::text;
        END IF;

        -- If ALL conditions match (AND logic within scope), return TRUE
        IF tags_match AND agents_match AND names_match AND id_match THEN
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

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbook_runs') THEN
        EXECUTE 'ALTER TABLE playbook_runs ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'checks') THEN
        EXECUTE 'ALTER TABLE checks ENABLE ROW LEVEL SECURITY;';
    END IF;

    -- Another relation called "views" exists in the information_schema schema.
    IF NOT (SELECT c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE c.relname = 'views' AND n.nspname = 'public') THEN
        EXECUTE 'ALTER TABLE views ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'view_panels') THEN
        EXECUTE 'ALTER TABLE view_panels ENABLE ROW LEVEL SECURITY;';
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
          config_items.name,
          config_items.id
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
          components.name,
          components.id
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
          canaries.agent_id,
          canaries.name,
          canaries.id
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
          playbooks.name,
          playbooks.id
        )
      END
    );

-- Policy playbook_runs
DROP POLICY IF EXISTS playbook_runs_auth ON playbook_runs;

CREATE POLICY playbook_runs_auth ON playbook_runs
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE (
        -- User must have access to the playbook
        EXISTS (
          SELECT 1
          FROM playbooks
          WHERE playbooks.id = playbook_runs.playbook_id
        )
        AND
        -- AND if run has a config_id, user must have access to that config
        (playbook_runs.config_id IS NULL OR EXISTS (
          SELECT 1
          FROM config_items
          WHERE config_items.id = playbook_runs.config_id
        ))
        AND
        -- AND if run has a check_id, user must have access to that check (via its canary)
        (playbook_runs.check_id IS NULL OR EXISTS (
          SELECT 1
          FROM checks
          WHERE checks.id = playbook_runs.check_id
        ))
        -- Note: component_id check omitted (phasing out topology soon)
      )
      END
    );

-- Policy checks
DROP POLICY IF EXISTS checks_auth ON checks;

CREATE POLICY checks_auth ON checks
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on canaries
        SELECT 1
        FROM canaries
        WHERE canaries.id = checks.canary_id
      )
      END
    );

-- Policy views
DROP POLICY IF EXISTS views_auth ON views;

CREATE POLICY views_auth ON views
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        match_scope(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'view',
          NULL,
          NULL,
          views.name,
          views.id
        )
      END
    );

-- Policy view_panels (inherits from parent views table)
DROP POLICY IF EXISTS view_panels_auth ON view_panels;

CREATE POLICY view_panels_auth ON view_panels
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (
          SELECT 1 FROM views
          WHERE views.id = view_panels.view_id
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
ALTER VIEW views_summary SET (security_invoker = true);
