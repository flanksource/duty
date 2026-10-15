-- Function to check if user has access to a row on a generated table for a view based on grants
-- Grants are a jsonb array of scope UUIDs that are allowed to access the row
-- User scopes are provided in JWT claims
-- NULL or empty grants means row is hidden from all (no access)
--
-- Examples:
--   User scopes: ['scope-a', 'scope-b'], Row grants: ['scope-a', 'scope-c'] → TRUE (scope-a overlaps)
--   User scopes: ['scope-x'], Row grants: ['scope-a', 'scope-b'] → FALSE (no overlap)
--   User scopes: ['scope-a'], Row grants: NULL → FALSE (hidden)
--   User scopes: ['scope-a'], Row grants: [] → FALSE (hidden)
CREATE
OR REPLACE FUNCTION check_view_grants(grants jsonb) RETURNS BOOLEAN AS $$
BEGIN
  IF grants IS NULL OR jsonb_array_length(grants) = 0 THEN
    RETURN FALSE;
  END IF;

  RETURN EXISTS (
    SELECT 1 FROM jsonb_array_elements_text(grants) AS grant_uuid
    WHERE grant_uuid = ANY(
      COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(
          COALESCE(current_setting('request.jwt.claims', TRUE)::jsonb -> 'scopes', '[]'::jsonb)
        )), '{}'::text[]
      )
    )
  );
END;
$$ LANGUAGE plpgsql VOLATILE;
-- view_table_view_id returns the id of the one live view whose generated table is table_name, or NULL when no view
-- or more than one does. View names map to tables lossily ('a-b/c' and 'a/b-c' are both view_a_b_c), and a table
-- two views share holds rows of both, so neither view's grant can admit its rows. It reads views as its owner, so
-- the answer doesn't depend on which views the caller sees. The table name is the view's GeneratedTableName.
--
--   SELECT view_table_view_id('view_mc_pods');  => the id of view mc/pods
CREATE
OR REPLACE FUNCTION view_table_view_id(table_name text) RETURNS uuid AS $$
  SELECT CASE WHEN count(*) = 1 THEN (array_agg(id))[1] END
  FROM views
  WHERE deleted_at IS NULL
    AND 'view_' || replace(COALESCE(namespace, ''), '-', '_') || '_' || replace(name, '-', '_') = table_name
$$ LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp;

-- view_table_readable reports whether the caller may read the view whose generated table is table_name: it reads
-- that view as the caller, so the views table's policy decides. A readable view's rows are all readable, since a
-- view is granted whole. A table no single view owns is readable through no view.
--
--   claim {"view": [{"scope": "<Scope selecting view mc/pods>"}]}:
--   SELECT view_table_readable('view_mc_pods');  => true
CREATE
OR REPLACE FUNCTION view_table_readable(table_name text) RETURNS BOOLEAN AS $$
  SELECT EXISTS (SELECT 1 FROM views WHERE id = (SELECT view_table_view_id(table_name)))
$$ LANGUAGE sql STABLE SECURITY INVOKER;

-- ensure_view_table_policy enables row-level security on a generated view table and writes its read policy, unless
-- it's already current. Rows are only read through PostgREST: its roles get no policy for writes, so they can't
-- write. A row is readable when its view is, or when its __grants name a Scope in the claim's scopes, which only
-- Permissions grant.
--
--   SELECT ensure_view_table_policy('view_mc_pods');
CREATE
OR REPLACE FUNCTION ensure_view_table_policy(table_name text) RETURNS void AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
      WHERE n.nspname = 'public' AND c.relname = table_name AND c.relkind = 'r' AND c.relrowsecurity)
    AND EXISTS (SELECT 1 FROM pg_policies p
      WHERE p.schemaname = 'public' AND p.tablename = table_name AND p.policyname = 'view_grants_policy'
        AND p.cmd = 'SELECT' AND p.qual LIKE '%view_table_readable%') THEN
    RETURN;
  END IF;

  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
  EXECUTE format('DROP POLICY IF EXISTS view_grants_policy ON %I', table_name);
  EXECUTE format(
    'CREATE POLICY view_grants_policy ON %I FOR SELECT TO postgrest_api, postgrest_anon
      USING (CASE WHEN is_rls_disabled() THEN TRUE ELSE (SELECT view_table_readable(%L)) OR check_view_grants(__grants) END)',
    table_name, table_name);
END;
$$ LANGUAGE plpgsql;

-- Every generated view table gets the current policy when this script changes, whether or not its view refreshes again.
DO $$
DECLARE
  t text;
BEGIN
  FOR t IN
    SELECT c.table_name FROM information_schema.columns c
    WHERE c.table_schema = 'public' AND c.column_name = '__grants' AND c.table_name LIKE 'view\_%'
  LOOP
    PERFORM ensure_view_table_policy(t);
  END LOOP;
END
$$;
