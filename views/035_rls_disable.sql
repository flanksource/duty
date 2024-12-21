-- Disable RLS
ALTER TABLE config_items DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_changes DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_analysis DISABLE ROW LEVEL SECURITY;

ALTER TABLE components DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_component_relationships DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_relationships DISABLE ROW LEVEL SECURITY;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;

DROP POLICY IF EXISTS config_changes_auth ON config_changes;

DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

