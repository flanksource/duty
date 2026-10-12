-- Moves Scopes without a namespace into 'default', ahead of the schema apply that makes scopes.namespace NOT NULL.
-- They never go to '': that namespace is reserved for built-in Scopes.
--
-- A Scope already named 'default/<name>', or a second NULL-namespace Scope of the same name, would collide on
-- scopes_namespace_name_idx. Of the live Scopes that end up as 'default/<name>' one is kept: the one already in
-- 'default', else the most recently updated. The rest are soft deleted, and their membership cleared as
-- membership.Clear does for any soft-deleted Scope.
DO $$
DECLARE
  duplicates uuid[];
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'scopes' AND column_name = 'namespace' AND is_nullable = 'YES'
  ) THEN
    RETURN;
  END IF;

  WITH ranked AS (
    SELECT
      id,
      namespace,
      ROW_NUMBER() OVER (
        PARTITION BY name
        ORDER BY (namespace IS NOT NULL) DESC, updated_at DESC, created_at DESC, id DESC
      ) AS row_number
    FROM scopes
    WHERE deleted_at IS NULL AND (namespace IS NULL OR namespace = 'default')
  )
  SELECT COALESCE(array_agg(id), '{}') INTO duplicates FROM ranked WHERE namespace IS NULL AND row_number > 1;

  UPDATE scopes SET deleted_at = NOW() WHERE id = ANY(duplicates);

  IF to_regclass('public.scope_targets') IS NOT NULL THEN
    DELETE FROM scope_targets WHERE scope_id = ANY(duplicates);
  END IF;
  IF to_regclass('public.scope_members') IS NOT NULL THEN
    DELETE FROM scope_members WHERE scope_id = ANY(duplicates);
  END IF;

  UPDATE scopes SET namespace = 'default' WHERE namespace IS NULL;
END $$;
