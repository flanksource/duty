-- dependsOn: views/050_scope_membership_helpers.sql

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
    SELECT * INTO cols FROM _scope_resource_columns(kind, 'r');
    desired := NULL;
    FOR tgt IN SELECT * FROM scope_targets st WHERE st.scope_id = scope AND st.resource_type = kind
    LOOP
      desired := concat_ws(' UNION ', desired, format(
        'SELECT r.id FROM %s r WHERE _scope_target_matches(%L::uuid, %L::text, %L::text, %L::text, %L::uuid, %L::text[], %L::jsonb, %L::jsonb, %s, %s, %s, %s, %s, %s, %s)',
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

-- scope_contains reports whether a resource is in the Scope: the Scope has a member row for it, or one for its whole
-- type (resource_id NULL). A Scope with neither contains nothing. Row-level security reads membership through it.
--
--   SELECT scope_contains($scope, 'config', $id);  => true if ($scope, config, $id) or ($scope, config, NULL) exists
CREATE OR REPLACE FUNCTION scope_contains(scope uuid, kind text, resource_id uuid)
  RETURNS boolean
  AS $$
  SELECT EXISTS (
    SELECT 1
    FROM scope_members m
    WHERE m.scope_id = scope
      AND m.resource_type = kind
      AND (m.resource_id = scope_contains.resource_id OR m.resource_id IS NULL)
  )
$$
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp;

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
LANGUAGE plpgsql SET lock_timeout = '1s';

CREATE OR REPLACE TRIGGER scopes_delete_membership
  AFTER DELETE ON scopes REFERENCING OLD TABLE AS old_rows
  FOR EACH STATEMENT EXECUTE FUNCTION delete_scope_membership();

-- For each resource type, generate three trigger functions that keep scope_members in step with writes to its table,
-- in the writing transaction. They take the membership lock shared, so a Scope's rebuild, which takes it exclusively,
-- never misses a write. Setting deleted_at changes nothing.
--
-- They run as their definer so any writer can store membership, and pin search_path so the names in their bodies
-- resolve to the public schema whatever the caller's path says. scope_contains does the same.
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
    SELECT * INTO c FROM _scope_resource_columns(def.kind, 'r');
    match_targets := format(
      'SELECT t.scope_id, r.id AS resource_id FROM %%s r JOIN scope_targets t
         ON t.resource_type = %1$L AND t.lookup_keys && _scope_lookup_keys(%2$s, %3$s, %4$s)
        AND _scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels, %5$s, %6$s, %4$s, %7$s, %8$s, %2$s, %3$s)
       UNION
       SELECT t.scope_id, r.id FROM %%s r JOIN scope_targets t
         ON t.resource_type = %1$L AND t.lookup_keys = ''{}''::text[]
        AND _scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags, t.labels, %5$s, %6$s, %4$s, %7$s, %8$s, %2$s, %3$s)',
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
      LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp;
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
      LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp;
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
      LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp;
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

-- Registering, renaming or deleting an agent can change what a Scope's agent resolves to, so Mission Control
-- re-validates every Scope. Heartbeats and other updates don't notify.
CREATE OR REPLACE TRIGGER agents_scope_validity
  AFTER INSERT OR DELETE ON agents
  FOR EACH ROW EXECUTE PROCEDURE notify_table_updates_and_deletes();

CREATE OR REPLACE TRIGGER agents_scope_validity_update
  AFTER UPDATE OF name, deleted_at ON agents
  FOR EACH ROW
  WHEN (OLD.name IS DISTINCT FROM NEW.name OR OLD.deleted_at IS DISTINCT FROM NEW.deleted_at)
  EXECUTE PROCEDURE notify_table_updates_and_deletes();

-- rls_grants_admit reports whether the request's claim grants a row of the given type:
-- the claim grants "all" rows of the type, or the row is in every Scope of at least one grant.
-- A type without grants, or a grant naming a Scope with no membership rows, admits nothing.
CREATE OR REPLACE FUNCTION rls_grants_admit(kind text, row_id uuid)
  RETURNS boolean
  AS $$
  SELECT CASE
    WHEN c IS NULL THEN FALSE
    WHEN c = '"all"'::jsonb THEN TRUE
    WHEN jsonb_typeof(c) <> 'array' THEN FALSE
    ELSE EXISTS (
      SELECT 1
      FROM jsonb_array_elements(c) AS g(scopes)
      WHERE jsonb_typeof(g.scopes) = 'array'
        AND jsonb_array_length(g.scopes) > 0
        AND NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements_text(g.scopes) AS s(scope_id)
          WHERE NOT scope_contains(s.scope_id::uuid, kind, row_id)
        )
    )
  END
  FROM (SELECT current_setting('request.jwt.claims', TRUE)::jsonb -> kind AS c) AS claim
$$
LANGUAGE sql STABLE;

-- rls_grants_admit_row reports whether the request's claim grants a row with the given values, as rls_grants_admit
-- does, but matches the values against each Scope's targets instead of reading stored membership. Row-level security
-- checks a new or updated row with it: the row is checked before the membership triggers match it, so its stored
-- membership is still that of the old row, or none.
--
--   claim {"config": [["<scope with target tags cluster=demo>"]]}:
--   SELECT rls_grants_admit_row('config', $id, 'web', NULL, NULL, 'Kubernetes::Pod', '{"cluster":"aws"}', NULL);  => false
CREATE OR REPLACE FUNCTION rls_grants_admit_row(
  kind text, r_id uuid, r_name text, r_namespace text, r_agent_id uuid, r_type text, r_tags jsonb, r_labels jsonb)
  RETURNS boolean
  AS $$
  SELECT CASE
    WHEN c IS NULL THEN FALSE
    WHEN c = '"all"'::jsonb THEN TRUE
    WHEN jsonb_typeof(c) <> 'array' THEN FALSE
    ELSE EXISTS (
      SELECT 1
      FROM jsonb_array_elements(c) AS g(scopes)
      WHERE jsonb_typeof(g.scopes) = 'array'
        AND jsonb_array_length(g.scopes) > 0
        AND NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements_text(g.scopes) AS s(scope_id)
          WHERE NOT EXISTS (
              SELECT 1 FROM scope_members m
              WHERE m.scope_id = s.scope_id::uuid AND m.resource_type = kind AND m.resource_id IS NULL
            )
            AND NOT EXISTS (
              SELECT 1 FROM scope_targets t
              WHERE t.scope_id = s.scope_id::uuid AND t.resource_type = kind
                AND _scope_target_matches(t.resource_id, t.name, t.name_prefix, t.namespace, t.agent_id, t.types, t.tags,
                  t.labels, r_id, r_name, r_namespace, r_agent_id, r_type, r_tags, r_labels)
            )
        )
    )
  END
  FROM (SELECT current_setting('request.jwt.claims', TRUE)::jsonb -> kind AS c) AS claim
$$
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp;
