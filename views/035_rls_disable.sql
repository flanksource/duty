ALTER TABLE config_items DISABLE ROW LEVEL SECURITY;

ALTER TABLE components DISABLE ROW LEVEL SECURITY;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;

-- View owners
ALTER VIEW config_detail OWNER TO current_user;
ALTER VIEW config_summary OWNER TO current_user;
ALTER VIEW config_labels OWNER TO current_user;
ALTER VIEW config_names OWNER TO current_user;
ALTER VIEW config_statuses OWNER TO current_user;

