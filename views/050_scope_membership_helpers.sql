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

-- _scope_matches reports whether a resource with the given values belongs to the Scope by its targets: the Scope
-- selects the resource's whole type, or one of its targets matches the values. It decides what the resource's
-- membership will be once the triggers match it.
--
--   a Scope whose only target is config tags env=prod:
--   SELECT _scope_matches($scope, 'config', $id, 'web', NULL, NULL, NULL, '{"env":"prod"}', NULL);  => true
CREATE OR REPLACE FUNCTION _scope_matches(
  scope uuid, kind text, r_id uuid, r_name text, r_namespace text, r_agent_id uuid, r_type text, r_tags jsonb, r_labels jsonb)
  RETURNS boolean
  AS $$
  SELECT EXISTS (
      SELECT 1 FROM scope_members m
      WHERE m.scope_id = scope AND m.resource_type = kind AND m.resource_id IS NULL
    )
    OR EXISTS (
      SELECT 1 FROM scope_targets t
      WHERE t.scope_id = scope AND t.resource_type = kind
        AND _scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels,
          r_id, r_name, r_namespace, r_agent_id, r_type, r_tags, r_labels)
    )
$$
LANGUAGE sql STABLE;

-- _rls_claim returns the request's claim for a resource type: "all", a list of grants, or NULL when it has none.
--
--   with request.jwt.claims {"config": "all"}:  SELECT _rls_claim('config');  => "all"
CREATE OR REPLACE FUNCTION _rls_claim(kind text)
  RETURNS jsonb
  AS $$
  SELECT current_setting('request.jwt.claims', TRUE)::jsonb -> kind
$$
LANGUAGE sql STABLE;

-- _rls_grant_scopes returns the Scopes a grant of the claim requires a row to be in: its scope, its constraint, and
-- each of its impersonated Scopes. It returns NULL for a grant that isn't well formed, i.e. not an object, naming no
-- Scope, with a constraint but no scope, or with a value that isn't a UUID, so the grant admits nothing.
--
--   SELECT _rls_grant_scopes('{"scope": "<a>", "constraint": "<b>", "impersonated": ["<c>"]}');  => {<a>,<b>,<c>}
--   SELECT _rls_grant_scopes('{"constraint": "<b>"}');                                          => NULL
CREATE OR REPLACE FUNCTION _rls_grant_scopes(item jsonb)
  RETURNS uuid[]
  AS $$
DECLARE
  ids text[] := '{}';
  value jsonb;
BEGIN
  IF jsonb_typeof(item) IS DISTINCT FROM 'object' THEN
    RETURN NULL;
  END IF;

  IF item ? 'scope' THEN
    IF jsonb_typeof(item->'scope') <> 'string' THEN
      RETURN NULL;
    END IF;
    ids := ids || (item->>'scope');
  END IF;

  IF item ? 'constraint' THEN
    IF jsonb_typeof(item->'constraint') <> 'string' OR NOT item ? 'scope' THEN
      RETURN NULL;
    END IF;
    ids := ids || (item->>'constraint');
  END IF;

  IF item ? 'impersonated' THEN
    IF jsonb_typeof(item->'impersonated') <> 'array' THEN
      RETURN NULL;
    END IF;
    FOR value IN SELECT jsonb_array_elements(item->'impersonated') LOOP
      IF jsonb_typeof(value) <> 'string' THEN
        RETURN NULL;
      END IF;
      ids := ids || (value #>> '{}');
    END LOOP;
  END IF;

  IF cardinality(ids) = 0
    OR EXISTS (SELECT 1 FROM unnest(ids) AS v WHERE v !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$') THEN
    RETURN NULL;
  END IF;
  RETURN ids::uuid[];
END;
$$
LANGUAGE plpgsql IMMUTABLE;
