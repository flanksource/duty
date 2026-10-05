-- Scope membership, stored synchronously (mission-control specs/authorization/design/materialised-membership.md).
--
-- A Scope's targets are rows of scope_targets. scope_target_matches is the one predicate deciding whether a resource
-- matches a target, used both ways: a written resource against every target of its type (the triggers below), and a
-- saved Scope's targets against every resource (scope_rebuild_members). Membership is stored in scope_members.

-- scope_match_keys returns the tag, label and namespace conditions of a target, or the tags, labels and namespace of
-- a resource, as keys. A resource can only match a target whose keys are all among its own.
CREATE OR REPLACE FUNCTION scope_match_keys(tags jsonb, labels jsonb, namespace text)
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

CREATE OR REPLACE FUNCTION scope_targets_set_match_keys()
  RETURNS TRIGGER
  AS $$
BEGIN
  NEW.match_keys := scope_match_keys(NEW.tags, NEW.labels, NEW.namespace);
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER scope_targets_match_keys
  BEFORE INSERT OR UPDATE ON scope_targets
  FOR EACH ROW EXECUTE FUNCTION scope_targets_set_match_keys();

-- scope_target_matches reports whether a resource matches a target: every condition the target sets holds.
-- Names and namespaces match case-sensitively, and a prefix is compared literally.
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

-- scope_resource_columns returns the table of a resource type whose membership is stored, and the expressions of its
-- fields a target can select, over the row alias, in scope_target_matches' order. A field the type doesn't have is NULL.
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

-- Triggers matching written resources against the targets of their type, in the writing transaction.
-- They take the membership lock shared, so a Scope's rebuild, which takes it exclusively, never misses a write.
-- Inserts and hard deletes are handled once per statement. Updates are matched only when a field a Scope can select
-- changes, decided by the trigger's WHEN clause. Setting deleted_at changes nothing.
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
         ON t.resource_type = %1$L AND t.match_keys && scope_match_keys(%2$s, %3$s, %4$s)
        AND scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels, %5$s, %6$s, %4$s, %7$s, %8$s, %2$s, %3$s)
       UNION
       SELECT t.scope_id, r.id FROM %%s r JOIN scope_targets t
         ON t.resource_type = %1$L AND t.match_keys = ''{}''::text[]
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

-- scope_rebuild_members makes the Scope's stored members of every type it has targets of exactly the resources its
-- targets match, writing only the rows that change. Each target's values are constants in the query, so the tables'
-- indexes apply. Members of types it has neither targets nor a whole-type row for are deleted. The caller holds the
-- membership lock exclusively.
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

-- scope_admits reports whether a row of the given type is in the Scope: it has a membership row for the row,
-- or one for the whole type. A Scope with neither admits nothing.
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

-- A Scope deleted outright loses its targets and members. A soft delete is handled where the Scope is saved.
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
