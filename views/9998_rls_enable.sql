-- runs: always
-- Runs on every migration so that RLS is enabled even on databases that last migrated with RLS disabled.
-- Keep it cheap and lock-free when nothing changed: the policies live in 9997_rls_policies.sql, which runs only when it changes.

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
