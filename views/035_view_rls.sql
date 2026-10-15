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
-- view_table_readable reports whether the caller may read the view whose generated table is table_name. It reads
-- views as the caller, so the views table's policy decides. A readable view's rows are all readable: a view is
-- granted whole. The table name is the view's GeneratedTableName.
--
--   claim {"view": [{"scope": "<Scope selecting view mc/pods>"}]}:
--   SELECT view_table_readable('view_mc_pods');  => true
CREATE
OR REPLACE FUNCTION view_table_readable(table_name text) RETURNS BOOLEAN AS $$
  SELECT EXISTS (
    SELECT 1 FROM views
    WHERE deleted_at IS NULL
      AND 'view_' || replace(COALESCE(namespace, ''), '-', '_') || '_' || replace(name, '-', '_') = table_name
  )
$$ LANGUAGE sql STABLE SECURITY INVOKER;
