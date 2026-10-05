-- Internal helpers for 051_scope_membership.sql. Nothing outside the scope membership scripts calls them.

-- _scope_lookup_keys flattens tags, labels and a namespace into one array of keys. For a target the keys are its
-- conditions, for a resource its attributes, so a resource can only match a target whose keys are all among its own.
--
--   SELECT _scope_lookup_keys('{"env":"prod"}', '{"team":"x"}', 'default');
--   => {tag:env=prod,label:team=x,ns:default}
CREATE OR REPLACE FUNCTION _scope_lookup_keys(tags jsonb, labels jsonb, namespace text)
  RETURNS text[]
  AS $$
  SELECT ARRAY(
    SELECT 'tag:' || key || '=' || value
    FROM jsonb_each_text(CASE WHEN jsonb_typeof(tags) = 'object' THEN tags ELSE '{}'::jsonb END)
    UNION ALL
    SELECT 'label:' || key || '=' || value
    FROM jsonb_each_text(CASE WHEN jsonb_typeof(labels) = 'object' THEN labels ELSE '{}'::jsonb END)
    UNION ALL
    SELECT 'ns:' || namespace WHERE namespace IS NOT NULL
  )
$$
LANGUAGE sql IMMUTABLE PARALLEL SAFE;

-- _scope_targets_set_lookup_keys fills scope_targets.lookup_keys from the row's tags, labels and namespace before it is
-- written, so callers never set it.
--
--   INSERT INTO scope_targets (scope_id, resource_type, tags) VALUES ($1, 'config', '{"env":"prod"}');
--   => lookup_keys = {tag:env=prod}
CREATE OR REPLACE FUNCTION _scope_targets_set_lookup_keys()
  RETURNS TRIGGER
  AS $$
BEGIN
  NEW.lookup_keys := _scope_lookup_keys(NEW.tags, NEW.labels, NEW.namespace);
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER scope_targets_lookup_keys
  BEFORE INSERT OR UPDATE ON scope_targets
  FOR EACH ROW EXECUTE FUNCTION _scope_targets_set_lookup_keys();

-- _scope_target_matches reports whether a resource matches a target: every condition the target sets (t_*) holds for
-- the resource (r_*), and a NULL condition matches anything. Names and namespaces match case-sensitively, and a prefix
-- is compared literally.
--
--   target name_prefix 'web-', tags {"env":"prod"}; resource name 'web-1', tags {"env":"prod","app":"a"} => true
--   target name_prefix 'web-', tags {"env":"prod"}; resource name 'api-1', tags {"env":"prod"}         => false
CREATE OR REPLACE FUNCTION _scope_target_matches(
  t_resource_id uuid, t_name text, t_name_prefix text, t_namespace text, t_agent_id uuid, t_types text[], t_tags jsonb, t_labels jsonb,
  r_id uuid, r_name text, r_namespace text, r_agent_id uuid, r_type text, r_tags jsonb, r_labels jsonb)
  RETURNS boolean
  AS $$
  SELECT COALESCE(
    (t_resource_id IS NULL OR r_id = t_resource_id)
    AND (t_name IS NULL OR r_name = t_name)
    AND (t_name_prefix IS NULL OR starts_with(r_name, t_name_prefix))
    AND (t_namespace IS NULL OR r_namespace = t_namespace)
    AND (t_agent_id IS NULL OR r_agent_id = t_agent_id)
    AND (t_types IS NULL OR r_type = ANY (t_types))
    AND (t_tags IS NULL OR r_tags @> t_tags)
    AND (t_labels IS NULL OR r_labels @> t_labels),
    FALSE)
$$
LANGUAGE sql IMMUTABLE PARALLEL SAFE;

-- _scope_resource_columns returns, for a resource type, its table and the SQL expressions of the fields a target can
-- select, over the given row alias, in _scope_target_matches' order. A field the type doesn't have is NULL. It is used
-- to generate the trigger functions and the rebuild query for each type.
--
--   SELECT * FROM _scope_resource_columns('canary', 'r');
--   => tbl canaries, id r.id, name r.name, namespace r.namespace, agent_id r.agent_id, type NULL::text,
--      tags NULL::jsonb, labels r.labels
CREATE OR REPLACE FUNCTION _scope_resource_columns(kind text, alias text)
  RETURNS TABLE (tbl text, id text, name text, namespace text, agent_id text, type text, tags text, labels text)
  AS $$
BEGIN
  CASE kind
  WHEN 'config' THEN
    RETURN QUERY SELECT 'config_items', alias || '.id', alias || '.name', '(' || alias || '.tags->>''namespace'')',
      alias || '.agent_id', alias || '.type', alias || '.tags', alias || '.labels';
  WHEN 'component' THEN
    RETURN QUERY SELECT 'components', alias || '.id', alias || '.name', alias || '.namespace',
      alias || '.agent_id', alias || '.type', 'NULL::jsonb', alias || '.labels';
  WHEN 'check' THEN
    RETURN QUERY SELECT 'checks', alias || '.id', alias || '.name', alias || '.namespace',
      alias || '.agent_id', alias || '.type', 'NULL::jsonb', alias || '.labels';
  WHEN 'canary' THEN
    RETURN QUERY SELECT 'canaries', alias || '.id', alias || '.name', alias || '.namespace',
      alias || '.agent_id', 'NULL::text', 'NULL::jsonb', alias || '.labels';
  WHEN 'playbook' THEN
    RETURN QUERY SELECT 'playbooks', alias || '.id', alias || '.name', alias || '.namespace',
      'NULL::uuid', 'NULL::text', 'NULL::jsonb', 'NULL::jsonb';
  WHEN 'connection' THEN
    RETURN QUERY SELECT 'connections', alias || '.id', alias || '.name', alias || '.namespace',
      'NULL::uuid', alias || '.type', 'NULL::jsonb', 'NULL::jsonb';
  ELSE
    RAISE EXCEPTION 'membership of % isn''t stored', kind;
  END CASE;
END;
$$
LANGUAGE plpgsql IMMUTABLE;
