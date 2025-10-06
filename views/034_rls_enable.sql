CREATE
OR REPLACE FUNCTION is_rls_disabled () RETURNS BOOLEAN AS $$
BEGIN
  RETURN (current_setting('request.jwt.claims', TRUE) IS NULL
    OR current_setting('request.jwt.claims', TRUE) = ''
    OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Helper function to check if a name matches a selector pattern
-- Supports comma-separated names and wildcards (prefix: "name-*", suffix: "*-name")
CREATE OR REPLACE FUNCTION matches_name_selector(name TEXT, selector TEXT) RETURNS BOOLEAN AS $$
DECLARE
  pattern TEXT;
  patterns TEXT[];
BEGIN
  IF selector IS NULL OR name IS NULL THEN
    RETURN FALSE;
  END IF;

  -- Split comma-separated patterns
  patterns := string_to_array(selector, ',');

  FOREACH pattern IN ARRAY patterns LOOP
    pattern := trim(pattern);

    -- Exact match
    IF pattern = name THEN
      RETURN TRUE;
    END IF;

    -- Prefix wildcard: "echo-*" matches "echo-config", "echo-test", etc.
    IF pattern LIKE '%*' AND NOT pattern LIKE '*%*' THEN
      IF name LIKE (replace(pattern, '*', '%')) THEN
        RETURN TRUE;
      END IF;
    END IF;

    -- Suffix wildcard: "*-deployment" matches "restart-deployment", "scale-deployment", etc.
    IF pattern LIKE '*%' AND NOT pattern LIKE '%*%*' THEN
      IF name LIKE (replace(pattern, '*', '%')) THEN
        RETURN TRUE;
      END IF;
    END IF;
  END LOOP;

  RETURN FALSE;
END;
$$ LANGUAGE plpgsql IMMUTABLE SECURITY DEFINER;

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
      ELSE (
        agent_id = ANY (ARRAY (SELECT (jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'agents'))::uuid))
      )
      END
    );

-- Policy canaries
DROP POLICY IF EXISTS canaries_auth ON canaries;

CREATE POLICY canaries_auth ON canaries
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        SELECT 1 FROM jsonb_array_elements((current_setting('request.jwt.claims', TRUE)::json ->> 'labels')::jsonb) allowed_labels
        WHERE canaries.labels::jsonb @> allowed_labels.value
      )
      END
    );

-- Policy playbooks
DROP POLICY IF EXISTS playbooks_auth ON playbooks;

CREATE POLICY playbooks_auth ON playbooks
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE (
        -- Check if 'playbooks' object type is allowed
        'playbooks' = ANY (ARRAY(SELECT jsonb_array_elements_text(current_setting('request.jwt.claims')::jsonb -> 'objects')))
        OR
        -- Check if playbook name matches any selector in object_selectors.playbooks
        EXISTS (
          SELECT 1
          FROM jsonb_array_elements(current_setting('request.jwt.claims')::jsonb -> 'object_selectors' -> 'playbooks') AS selector
          WHERE matches_name_selector(playbooks.name, selector ->> 'name')
        )
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

-- TODO: Move 034_rls_enable.sql as the last script (eg: 10000_rls_enable.sql)
-- So that all the views are already created before it runs.