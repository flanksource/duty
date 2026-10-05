-- scope_lookup_keys flattens tags, labels and a namespace into one array of keys. For a target the keys are its
-- conditions, for a resource its attributes, so a resource can only match a target whose keys are all among its own.
--
--   SELECT scope_lookup_keys('{"env":"prod"}', '{"team":"x"}', 'default');
--   => {tag:env=prod,label:team=x,ns:default}
CREATE OR REPLACE FUNCTION scope_lookup_keys(tags jsonb, labels jsonb, namespace text)
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

-- scope_targets_set_lookup_keys fills scope_targets.lookup_keys from the row's tags, labels and namespace before it is
-- written, so callers never set it.
--
--   INSERT INTO scope_targets (scope_id, resource_type, tags) VALUES ($1, 'config', '{"env":"prod"}');
--   => lookup_keys = {tag:env=prod}
CREATE OR REPLACE FUNCTION scope_targets_set_lookup_keys()
  RETURNS TRIGGER
  AS $$
BEGIN
  NEW.lookup_keys := scope_lookup_keys(NEW.tags, NEW.labels, NEW.namespace);
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER scope_targets_lookup_keys
  BEFORE INSERT OR UPDATE ON scope_targets
  FOR EACH ROW EXECUTE FUNCTION scope_targets_set_lookup_keys();

-- scope_target_matches reports whether a resource matches a target: every condition the target sets (t_*) holds for
-- the resource (r_*), and a NULL condition matches anything. Names and namespaces match case-sensitively, and a prefix
-- is compared literally.
--
--   target name_prefix 'web-', tags {"env":"prod"}; resource name 'web-1', tags {"env":"prod","app":"a"} => true
--   target name_prefix 'web-', tags {"env":"prod"}; resource name 'api-1', tags {"env":"prod"}         => false
CREATE OR REPLACE FUNCTION scope_target_matches(
  t_id uuid, t_name text, t_name_prefix text, t_namespace text, t_agent_id uuid, t_types text[], t_tags jsonb, t_labels jsonb,
  r_id uuid, r_name text, r_namespace text, r_agent_id uuid, r_type text, r_tags jsonb, r_labels jsonb)
  RETURNS boolean
  AS $$
  SELECT COALESCE(
    (t_id IS NULL OR r_id = t_id)
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

-- scope_resource_columns returns, for a resource type, its table and the SQL expressions of the fields a target can
-- select, over the given row alias, in scope_target_matches' order. A field the type doesn't have is NULL. It is used
-- to generate the trigger functions and the rebuild query for each type.
--
--   SELECT * FROM scope_resource_columns('canary', 'r');
--   => tbl canaries, id r.id, name r.name, namespace r.namespace, agent_id r.agent_id, type NULL::text,
--      tags NULL::jsonb, labels r.labels
CREATE OR REPLACE FUNCTION scope_resource_columns(kind text, alias text)
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

-- For each resource type, generate three trigger functions that keep scope_members in step with writes to its table,
-- in the writing transaction. They take the membership lock shared, so a Scope's rebuild, which takes it exclusively,
-- never misses a write. Setting deleted_at changes nothing.
--
--   scope_members_inserted_<kind>: per statement, adds a member row for every inserted resource and every target it
--     matches. E.g. inserting config 'web-1' with tags {"env":"prod"} adds it to each Scope with a config target
--     tags env=prod.
--   scope_members_updated_<kind>: per row, only when a field a target can select changes (the trigger's WHEN clause),
--     deletes the resource's member rows and re-adds those it now matches. E.g. retagging 'web-1' to env=dev removes
--     it from the env=prod Scopes.
--   scope_members_deleted_<kind>: per statement, deletes the member rows of hard-deleted resources.
DO $$
DECLARE
  def record;
  c record;
  match_targets text;
BEGIN
  FOR def IN
    SELECT * FROM (VALUES
      ('config', 'config_items', 'id, name, type, agent_id, tags, labels'),
      ('component', 'components', 'id, name, namespace, type, agent_id, labels'),
      ('check', 'checks', 'id, name, namespace, type, agent_id, labels'),
      ('canary', 'canaries', 'id, name, namespace, agent_id, labels'),
      ('playbook', 'playbooks', 'id, name, namespace'),
      ('connection', 'connections', 'id, name, namespace, type')
    ) AS d(kind, tbl, watched)
  LOOP
    -- the targets of the type a resource row "r" matches: those sharing one of its keys, found through the GIN index,
    -- and those with no keys
    SELECT * INTO c FROM scope_resource_columns(def.kind, 'r');
    match_targets := format(
      'SELECT t.scope_id, r.id AS resource_id FROM %%s r JOIN scope_targets t
         ON t.resource_type = %1$L AND t.lookup_keys && scope_lookup_keys(%2$s, %3$s, %4$s)
        AND scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels, %5$s, %6$s, %4$s, %7$s, %8$s, %2$s, %3$s)
       UNION
       SELECT t.scope_id, r.id FROM %%s r JOIN scope_targets t
         ON t.resource_type = %1$L AND t.lookup_keys = ''{}''::text[]
        AND scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels, %5$s, %6$s, %4$s, %7$s, %8$s, %2$s, %3$s)',
      def.kind, c.tags, c.labels, c.namespace, c.id, c.name, c.agent_id, c.type);

    EXECUTE format($f$
      CREATE OR REPLACE FUNCTION scope_members_inserted_%1$s()
        RETURNS TRIGGER
        AS $body$
      BEGIN
        PERFORM pg_advisory_xact_lock_shared(hashtext('scope_membership'));
        INSERT INTO scope_members (scope_id, resource_type, resource_id)
        SELECT m.scope_id, %2$L, m.resource_id FROM (%3$s) m
        ON CONFLICT (scope_id, resource_type, resource_id) WHERE resource_id IS NOT NULL DO NOTHING;
        RETURN NULL;
      END;
      $body$
      LANGUAGE plpgsql SECURITY DEFINER;
    $f$, replace(def.kind, '-', '_'), def.kind, format(match_targets, 'new_rows', 'new_rows'));

    EXECUTE format($f$
      CREATE OR REPLACE FUNCTION scope_members_updated_%1$s()
        RETURNS TRIGGER
        AS $body$
      BEGIN
        PERFORM pg_advisory_xact_lock_shared(hashtext('scope_membership'));
        DELETE FROM scope_members WHERE resource_type = %2$L AND resource_id IN (OLD.id, NEW.id);
        INSERT INTO scope_members (scope_id, resource_type, resource_id)
        SELECT m.scope_id, %2$L, m.resource_id FROM (%3$s) m
        ON CONFLICT (scope_id, resource_type, resource_id) WHERE resource_id IS NOT NULL DO NOTHING;
        RETURN NULL;
      END;
      $body$
      LANGUAGE plpgsql SECURITY DEFINER;
    $f$, def.kind, def.kind, format(match_targets, '(SELECT NEW.*)', '(SELECT NEW.*)'));

    EXECUTE format($f$
      CREATE OR REPLACE FUNCTION scope_members_deleted_%1$s()
        RETURNS TRIGGER
        AS $body$
      BEGIN
        PERFORM pg_advisory_xact_lock_shared(hashtext('scope_membership'));
        DELETE FROM scope_members m USING old_rows o WHERE m.resource_type = %2$L AND m.resource_id = o.id;
        RETURN NULL;
      END;
      $body$
      LANGUAGE plpgsql SECURITY DEFINER;
    $f$, def.kind, def.kind);

    EXECUTE format('CREATE OR REPLACE TRIGGER %1$s_scope_membership_insert
      AFTER INSERT ON %1$s REFERENCING NEW TABLE AS new_rows
      FOR EACH STATEMENT EXECUTE FUNCTION scope_members_inserted_%2$s()', def.tbl, def.kind);

    EXECUTE format('CREATE OR REPLACE TRIGGER %1$s_scope_membership_delete
      AFTER DELETE ON %1$s REFERENCING OLD TABLE AS old_rows
      FOR EACH STATEMENT EXECUTE FUNCTION scope_members_deleted_%2$s()', def.tbl, def.kind);

    EXECUTE format('CREATE OR REPLACE TRIGGER %1$s_scope_membership_update
      AFTER UPDATE OF %2$s ON %1$s
      FOR EACH ROW
      WHEN (%3$s)
      EXECUTE FUNCTION scope_members_updated_%4$s()',
      def.tbl, def.watched,
      (SELECT string_agg(format('OLD.%1$s IS DISTINCT FROM NEW.%1$s', trim(col)), ' OR ')
        FROM unnest(string_to_array(def.watched, ',')) AS col),
      def.kind);
  END LOOP;
END
$$;

-- scope_rebuild_members recomputes a Scope's member rows from its scope_targets after the targets change, writing only
-- the rows that change. Members of types it has neither targets nor a whole-type row for are deleted. Each target's
-- values are constants in the query, so the tables' indexes apply. The caller holds the membership lock exclusively
-- (membership.Rebuild).
--
--   a Scope whose only target is config name 'web-1', changed to config tags env=prod:
--   SELECT scope_rebuild_members($1);  => removes web-1's row unless it's tagged env=prod, adds every env=prod config
CREATE OR REPLACE FUNCTION scope_rebuild_members(scope uuid)
  RETURNS void
  AS $$
DECLARE
  kind text;
  cols record;
  tgt record;
  desired text;
BEGIN
  DELETE FROM scope_members m
  WHERE m.scope_id = scope AND m.resource_id IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM scope_targets st WHERE st.scope_id = scope AND st.resource_type = m.resource_type);

  FOR kind IN SELECT DISTINCT st.resource_type FROM scope_targets st WHERE st.scope_id = scope
  LOOP
    SELECT * INTO cols FROM scope_resource_columns(kind, 'r');
    desired := NULL;
    FOR tgt IN SELECT * FROM scope_targets st WHERE st.scope_id = scope AND st.resource_type = kind
    LOOP
      desired := concat_ws(' UNION ', desired, format(
        'SELECT r.id FROM %s r WHERE scope_target_matches(%L::uuid, %L::text, %L::text, %L::text, %L::uuid, %L::text[], %L::jsonb, %L::jsonb, %s, %s, %s, %s, %s, %s, %s)',
        cols.tbl, tgt.resource_id, tgt.name, tgt.name_prefix, tgt.namespace, tgt.agent_id, tgt.types, tgt.tags, tgt.labels,
        cols.id, cols.name, cols.namespace, cols.agent_id, cols.type, cols.tags, cols.labels));
    END LOOP;

    EXECUTE format(
      'WITH desired AS (%s),
       removed AS (
         DELETE FROM scope_members m
         WHERE m.scope_id = %L AND m.resource_type = %L AND m.resource_id IS NOT NULL
           AND NOT EXISTS (SELECT 1 FROM desired d WHERE d.id = m.resource_id)
       )
       INSERT INTO scope_members (scope_id, resource_type, resource_id)
       SELECT %L, %L, d.id FROM desired d
       ON CONFLICT (scope_id, resource_type, resource_id) WHERE resource_id IS NOT NULL DO NOTHING',
      desired, scope, kind, scope, kind);
  END LOOP;
END;
$$
LANGUAGE plpgsql;

-- scope_admits reports whether a resource is in the Scope: the Scope has a member row for it, or one for its whole
-- type (resource_id NULL). A Scope with neither admits nothing. Row-level security reads membership through it.
--
--   SELECT scope_admits($scope, 'config', $id);  => true if ($scope, config, $id) or ($scope, config, NULL) exists
CREATE OR REPLACE FUNCTION scope_admits(scope uuid, kind text, row_id uuid)
  RETURNS boolean
  AS $$
  SELECT EXISTS (
    SELECT 1
    FROM scope_members m
    WHERE m.scope_id = scope
      AND m.resource_type = kind
      AND (m.resource_id = row_id OR m.resource_id IS NULL)
  )
$$
LANGUAGE sql STABLE SECURITY DEFINER;

-- delete_scope_membership deletes the targets and members of Scopes deleted outright from scopes, so no member row
-- outlives its Scope. A soft delete is handled where the Scope is saved (membership.Clear).
--
--   DELETE FROM scopes WHERE id = $1;  => scope_targets and scope_members rows with scope_id $1 are deleted too
CREATE OR REPLACE FUNCTION delete_scope_membership()
  RETURNS TRIGGER
  AS $$
BEGIN
  PERFORM pg_advisory_xact_lock(hashtext('scope_membership'));
  DELETE FROM scope_targets t USING old_rows o WHERE t.scope_id = o.id;
  DELETE FROM scope_members m USING old_rows o WHERE m.scope_id = o.id;
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER scopes_delete_membership
  AFTER DELETE ON scopes REFERENCING OLD TABLE AS old_rows
  FOR EACH STATEMENT EXECUTE FUNCTION delete_scope_membership();

-- Registering, renaming or deleting an agent can change what a Scope's agent resolves to, so Mission Control
-- re-validates every Scope. Heartbeats and other updates don't notify.
DROP TRIGGER IF EXISTS agents_scope_validity ON agents;
DROP TRIGGER IF EXISTS agents_scope_validity_update ON agents;

CREATE OR REPLACE TRIGGER agents_scope_validity
  AFTER INSERT OR DELETE ON agents
  FOR EACH ROW EXECUTE PROCEDURE notify_table_updates_and_deletes();

CREATE OR REPLACE TRIGGER agents_scope_validity_update
  AFTER UPDATE OF name, deleted_at ON agents
  FOR EACH ROW
  WHEN (OLD.name IS DISTINCT FROM NEW.name OR OLD.deleted_at IS DISTINCT FROM NEW.deleted_at)
  EXECUTE PROCEDURE notify_table_updates_and_deletes();
