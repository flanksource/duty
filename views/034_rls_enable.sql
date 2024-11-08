ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;

ALTER TABLE components ENABLE ROW LEVEL SECURITY;

-- Policy config items
DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon
    USING (tags::jsonb @> (current_setting('request.jwt.claims', TRUE)::json ->> 'tags')::jsonb
      OR current_setting('request.jwt.claims', TRUE)::json -> 'agents' ? 'agent_id'::text);

DROP POLICY IF EXISTS config_items_view_owner_allow ON config_items;

CREATE POLICY config_items_view_owner_allow ON config_items
  FOR ALL TO api_views_owner
    USING (TRUE);

-- Policy components
DROP POLICY IF EXISTS components_auth ON components;

CREATE POLICY components_auth ON components
  FOR ALL TO postgrest_api, postgrest_anon
    USING (current_setting('request.jwt.claims', TRUE)::json -> 'agents' ? agent_id::text);

DROP POLICY IF EXISTS components_view_owner_allow ON components;

CREATE POLICY components_view_owner_allow ON components
  FOR ALL TO api_views_owner
    USING (TRUE);

-- TODO: Add more
ALTER VIEW config_detail OWNER TO api_views_owner;

ALTER VIEW config_labels OWNER TO api_views_owner;

ALTER VIEW config_names OWNER TO api_views_owner;

ALTER VIEW config_statuses OWNER TO api_views_owner;

ALTER VIEW config_summary OWNER TO api_views_owner;

