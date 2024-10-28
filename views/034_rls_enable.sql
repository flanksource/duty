ALTER TABLE config_items ENABLE ROW LEVEL SECURITY;

ALTER TABLE components ENABLE ROW LEVEL SECURITY;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

CREATE POLICY config_items_auth ON config_items
  FOR ALL TO postgrest_api, postgrest_anon
    USING (tags::jsonb @> (current_setting('request.jwt.claims', TRUE)::json ->> 'tags')::jsonb
      OR current_setting('request.jwt.claims', TRUE)::json ->> 'agent_id' = agent_id::text);

DROP POLICY IF EXISTS components_auth ON components;

CREATE POLICY components_auth ON components
  FOR ALL TO postgrest_api, postgrest_anon
    USING (current_setting('request.jwt.claims', TRUE)::json ->> 'agent_id' = agent_id::text);

