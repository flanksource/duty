CREATE
OR REPLACE FUNCTION is_rls_disabled () RETURNS BOOLEAN AS $$
BEGIN
  RETURN (current_setting('request.jwt.claims', TRUE) IS NULL
    OR current_setting('request.jwt.claims', TRUE) = ''
    OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Policy config items
ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;

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
ALTER TABLE config_changes ENABLE ROW LEVEL SECURITY;

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
ALTER TABLE config_analysis ENABLE ROW LEVEL SECURITY;

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
ALTER TABLE config_relationships ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS config_relationships_auth ON config_relationships;

CREATE POLICY config_relationships_auth ON config_relationships
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_relationships.config_id AND config_items.id = config_relationships.related_id
      )
      END
    );

-- Policy config_relationships
ALTER TABLE config_component_relationships ENABLE ROW LEVEL SECURITY;

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
ALTER TABLE components ENABLE ROW LEVEL SECURITY;

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

ALTER VIEW analysis_by_config SET (security_invoker = true);
ALTER VIEW catalog_changes SET (security_invoker = true);
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
ALTER VIEW incidents_by_config SET (security_invoker = true);