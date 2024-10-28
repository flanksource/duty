ALTER TABLE config_items DISABLE ROW LEVEL SECURITY;

ALTER TABLE components DISABLE ROW LEVEL SECURITY;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;
