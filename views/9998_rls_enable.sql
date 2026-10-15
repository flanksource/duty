-- runs: always
-- Runs on every migration so that RLS is enabled even on databases that last migrated with RLS disabled.

-- Views are filtered by their grants, like every resource table, so nothing matches rows against selectors.
DROP FUNCTION IF EXISTS match_scope(jsonb, jsonb, uuid, text, uuid);

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

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_costs') THEN
        EXECUTE 'ALTER TABLE config_costs ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_cost_compact') THEN
        EXECUTE 'ALTER TABLE config_cost_compact ENABLE ROW LEVEL SECURITY;';
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

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbook_run_actions') THEN
        EXECUTE 'ALTER TABLE playbook_run_actions ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbook_approvals') THEN
        EXECUTE 'ALTER TABLE playbook_approvals ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbook_action_agent_data') THEN
        EXECUTE 'ALTER TABLE playbook_action_agent_data ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'checks') THEN
        EXECUTE 'ALTER TABLE checks ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_scrapers') THEN
        EXECUTE 'ALTER TABLE config_scrapers ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'artifacts') THEN
        EXECUTE 'ALTER TABLE artifacts ENABLE ROW LEVEL SECURITY;';
    END IF;

    -- Another relation called "views" exists in the information_schema schema.
    IF NOT (SELECT c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE c.relname = 'views' AND n.nspname = 'public') THEN
        EXECUTE 'ALTER TABLE views ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE relname = 'view_panels') THEN
        EXECUTE 'ALTER TABLE view_panels ENABLE ROW LEVEL SECURITY;';
    END IF;

    -- Internal notification recovery queues have no public row policies.
    IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid = 'public.notification_health_states'::regclass) THEN
        EXECUTE 'ALTER TABLE public.notification_health_states ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid = 'public.notification_health_episodes'::regclass) THEN
        EXECUTE 'ALTER TABLE public.notification_health_episodes ENABLE ROW LEVEL SECURITY;';
    END IF;

    IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid = 'public.notification_deliveries'::regclass) THEN
        EXECUTE 'ALTER TABLE public.notification_deliveries ENABLE ROW LEVEL SECURITY;';
    END IF;
END $$;

-- Policy config items
DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (SELECT 1 FROM rls_matching_grants('config', config_items.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (config_items.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('config', config_items.id, config_items.name, config_items.tags->>'namespace', config_items.agent_id, config_items.type, config_items.tags, config_items.labels))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('config', config_items.id, config_items.name, config_items.tags->>'namespace', config_items.agent_id, config_items.type, config_items.tags, config_items.labels)
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

-- Policy config_costs
DROP POLICY IF EXISTS config_costs_auth ON config_costs;

CREATE POLICY config_costs_auth ON config_costs
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        -- just leverage the RLS on config_items
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_costs.config_id
      )
      END
    );

-- Policy config_cost_compact: identical rules to config_costs.
DROP POLICY IF EXISTS config_cost_compact_auth ON config_cost_compact;

CREATE POLICY config_cost_compact_auth ON config_cost_compact
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        SELECT 1
        FROM config_items
        WHERE config_items.id = config_cost_compact.config_id
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
        EXISTS (SELECT 1 FROM rls_matching_grants('component', components.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (components.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('component', components.id, components.name, components.namespace, components.agent_id, components.type, NULL, components.labels))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('component', components.id, components.name, components.namespace, components.agent_id, components.type, NULL, components.labels)
      END
    );

-- Policy canaries
DROP POLICY IF EXISTS canaries_auth ON canaries;

CREATE POLICY canaries_auth ON canaries
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (SELECT 1 FROM rls_matching_grants('canary', canaries.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (canaries.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('canary', canaries.id, canaries.name, canaries.namespace, canaries.agent_id, NULL, NULL, canaries.labels))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('canary', canaries.id, canaries.name, canaries.namespace, canaries.agent_id, NULL, NULL, canaries.labels)
      END
    );

-- Policy playbooks
DROP POLICY IF EXISTS playbooks_auth ON playbooks;

CREATE POLICY playbooks_auth ON playbooks
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (SELECT 1 FROM rls_matching_grants('playbook', playbooks.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (playbooks.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('playbook', playbooks.id, playbooks.name, playbooks.namespace, NULL, NULL, NULL, NULL))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('playbook', playbooks.id, playbooks.name, playbooks.namespace, NULL, NULL, NULL, NULL)
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

-- Policy playbook_run_actions (inherits from parent playbook_runs table)
DROP POLICY IF EXISTS playbook_run_actions_auth ON playbook_run_actions;

CREATE POLICY playbook_run_actions_auth ON playbook_run_actions
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (
          SELECT 1 FROM playbook_runs
          WHERE playbook_runs.id = playbook_run_actions.playbook_run_id
        )
      END
    );

-- Policy playbook_approvals (inherits from parent playbook_runs table)
DROP POLICY IF EXISTS playbook_approvals_auth ON playbook_approvals;

CREATE POLICY playbook_approvals_auth ON playbook_approvals
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (
          SELECT 1 FROM playbook_runs
          WHERE playbook_runs.id = playbook_approvals.run_id
        )
      END
    );

-- Policy playbook_action_agent_data (inherits from parent playbook_runs table)
DROP POLICY IF EXISTS playbook_action_agent_data_auth ON playbook_action_agent_data;

CREATE POLICY playbook_action_agent_data_auth ON playbook_action_agent_data
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (
          SELECT 1 FROM playbook_runs
          WHERE playbook_runs.id = playbook_action_agent_data.run_id
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
        -- a check is visible with its canary
        SELECT 1
        FROM canaries
        WHERE canaries.id = checks.canary_id
      ) OR EXISTS (SELECT 1 FROM rls_matching_grants('check', checks.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (checks.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('check', checks.id, checks.name, checks.namespace, checks.agent_id, checks.type, NULL, checks.labels))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE EXISTS (
        SELECT 1
        FROM canaries
        WHERE canaries.id = checks.canary_id
      ) OR rls_grants_admit_row('check', checks.id, checks.name, checks.namespace, checks.agent_id, checks.type, NULL, checks.labels)
      END
    );

-- Policy config_scrapers
DROP POLICY IF EXISTS config_scrapers_auth ON config_scrapers;

CREATE POLICY config_scrapers_auth ON config_scrapers
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        EXISTS (SELECT 1 FROM rls_matching_grants('scraper', config_scrapers.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (config_scrapers.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('scraper', config_scrapers.id, config_scrapers.name, config_scrapers.namespace, config_scrapers.agent_id, NULL, NULL, NULL))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('scraper', config_scrapers.id, config_scrapers.name, config_scrapers.namespace, config_scrapers.agent_id, NULL, NULL, NULL)
      END
    );

-- Policy artifacts: an artifact is visible with every producer it has: its playbook run's step, check, config change and scraper.
DROP POLICY IF EXISTS artifacts_auth ON artifacts;

CREATE POLICY artifacts_auth ON artifacts
  FOR ALL TO postgrest_api, postgrest_anon
    USING (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE (
        (artifacts.playbook_run_action_id IS NULL OR EXISTS (
          SELECT 1
          FROM playbook_run_actions
          WHERE playbook_run_actions.id = artifacts.playbook_run_action_id
        ))
        AND
        (artifacts.check_id IS NULL OR EXISTS (
          SELECT 1
          FROM checks
          WHERE checks.id = artifacts.check_id
        ))
        AND
        (artifacts.config_change_id IS NULL OR EXISTS (
          SELECT 1
          FROM config_changes
          WHERE config_changes.id = artifacts.config_change_id
        ))
        AND
        (artifacts.scraper_id IS NULL OR EXISTS (
          SELECT 1
          FROM config_scrapers
          WHERE config_scrapers.id = artifacts.scraper_id
        ))
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
        EXISTS (SELECT 1 FROM rls_matching_grants('view', views.id))
        -- The row an INSERT is adding, e.g. for its RETURNING, has no membership yet, so it's checked by its values. It's
        -- told from a stored row by its tuple id, which isn't set (no block, offset 0) until the row is stored.
        OR (views.ctid = '(4294967295,0)'::tid AND rls_grants_admit_row('view', views.id, views.name, views.namespace, NULL, NULL, NULL, NULL))
      END
    )
    WITH CHECK (
      CASE WHEN (SELECT is_rls_disabled()) THEN TRUE
      ELSE
        rls_grants_admit_row('view', views.id, views.name, views.namespace, NULL, NULL, NULL, NULL)
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
ALTER VIEW artifact_summary SET (security_invoker = true);
ALTER VIEW catalog_changes SET (security_invoker = true);
ALTER VIEW check_summary SET (security_invoker = true);
ALTER VIEW check_summary_by_config SET (security_invoker = true);
ALTER VIEW check_summary_for_config SET (security_invoker = true);
ALTER VIEW checks_by_config SET (security_invoker = true);
ALTER VIEW checks_labels_keys SET (security_invoker = true);
ALTER VIEW checks_status_artifacts SET (security_invoker = true);
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
ALTER VIEW external_group_summary SET (security_invoker = true);
ALTER VIEW topology SET (security_invoker = true);
ALTER VIEW incidents_by_config SET (security_invoker = true);
ALTER VIEW integrations_with_status SET (security_invoker = true);
ALTER VIEW playbook_names SET (security_invoker = true);
ALTER VIEW views_summary SET (security_invoker = true);
