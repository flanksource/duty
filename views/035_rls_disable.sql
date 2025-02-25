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
        EXECUTE 'ALTER TABLE config_changes DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'components') THEN
        EXECUTE 'ALTER TABLE components DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_component_relationships') THEN
        EXECUTE 'ALTER TABLE config_component_relationships DISABLE ROW LEVEL SECURITY;';
    END IF;

    IF (SELECT relrowsecurity FROM pg_class WHERE relname = 'config_relationships') THEN
        RAISE NOTICE 'RLS is already disabled on config_relationships.';
    END IF;
END $$;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;

DROP POLICY IF EXISTS config_changes_auth ON config_changes;

DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

