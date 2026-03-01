-- Disable RLS
DO $$
BEGIN
    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_items') THEN
        EXECUTE 'ALTER TABLE config_items DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_changes') THEN
        EXECUTE 'ALTER TABLE config_changes DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_analysis') THEN
        EXECUTE 'ALTER TABLE config_analysis DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'components') THEN
        EXECUTE 'ALTER TABLE components DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_component_relationships') THEN
        EXECUTE 'ALTER TABLE config_component_relationships DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_relationships') THEN
        EXECUTE 'ALTER TABLE config_relationships DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'canaries') THEN
        EXECUTE 'ALTER TABLE canaries DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbooks') THEN
        EXECUTE 'ALTER TABLE playbooks DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'playbook_runs') THEN
        EXECUTE 'ALTER TABLE playbook_runs DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'checks') THEN
        EXECUTE 'ALTER TABLE checks DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE c.relname = 'views' AND n.nspname = 'public') THEN
        EXECUTE 'ALTER TABLE views DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'view_panels') THEN
        EXECUTE 'ALTER TABLE view_panels DISABLE ROW LEVEL SECURITY;';
    END IF;

END $$;

-- Disable RLS on dynamically generated view tables
DO $$
DECLARE
    r record;
BEGIN
    FOR r IN
        SELECT c.relname, c.relrowsecurity
        FROM pg_class c
        JOIN pg_namespace n ON c.relnamespace = n.oid
        WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relname LIKE 'view\_%' ESCAPE '\'
    LOOP
        IF r.relrowsecurity THEN
            EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY;', r.relname);
        END IF;
    END LOOP;
END $$;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;

DROP POLICY IF EXISTS config_changes_auth ON config_changes;

DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

DROP POLICY IF EXISTS config_component_relationships_auth ON config_component_relationships;

DROP POLICY IF EXISTS config_relationships_auth ON config_relationships;

DROP POLICY IF EXISTS canaries_auth ON canaries;

DROP POLICY IF EXISTS playbooks_auth ON playbooks;

DROP POLICY IF EXISTS playbook_runs_auth ON playbook_runs;

DROP POLICY IF EXISTS checks_auth ON checks;

DROP POLICY IF EXISTS views_auth ON views;

DROP POLICY IF EXISTS view_panels_auth ON view_panels;

DO $$
DECLARE
    r record;
BEGIN
    -- Drop policies on dynamically generated view tables
    FOR r IN
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON c.relnamespace = n.oid
        WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relname LIKE 'view\_%' ESCAPE '\'
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS view_grants_policy ON %I;', r.relname);
    END LOOP;
END $$;
