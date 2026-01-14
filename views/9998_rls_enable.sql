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
        rls_has_wildcard('config')
        OR (COALESCE(config_items.__scope, '{}'::uuid[]) && rls_scope_access())
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
        rls_has_wildcard('component')
        OR (COALESCE(components.__scope, '{}'::uuid[]) && rls_scope_access())
      END
    );

-- Policy canaries
DROP POLICY IF EXISTS canaries_auth ON canaries;

CREATE POLICY canaries_auth ON canaries
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_has_wildcard('canary')
        OR (COALESCE(canaries.__scope, '{}'::uuid[]) && rls_scope_access())
      END
    );

-- Policy playbooks
DROP POLICY IF EXISTS playbooks_auth ON playbooks;

CREATE POLICY playbooks_auth ON playbooks
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_has_wildcard('playbook')
        OR (COALESCE(playbooks.__scope, '{}'::uuid[]) && rls_scope_access())
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
        rls_has_wildcard('view')
        OR (COALESCE(views.__scope, '{}'::uuid[]) && rls_scope_access())
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
