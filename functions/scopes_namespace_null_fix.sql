-- Moves Scopes with a NULL or '' namespace into 'default', ahead of the schema apply that requires every Scope
-- to have a non-empty namespace. Older releases had a nullable scopes.namespace.
--
-- A Scope already named 'default/<name>', or a second Scope of the same name without a namespace, would collide on
-- scopes_namespace_name_idx. Of the live Scopes that end up as 'default/<name>' one is kept: the one already in
-- 'default', else the most recently updated. The rest are soft deleted, and their membership cleared as
-- membership.Clear does for any soft-deleted Scope.
DO $$
DECLARE
  duplicates uuid[];
BEGIN
  IF to_regclass('public.scopes') IS NULL THEN
    RETURN;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM scopes WHERE namespace IS NULL OR namespace = '') THEN
    RETURN;
  END IF;

  WITH ranked AS (
    SELECT
      id,
      namespace,
      ROW_NUMBER() OVER (
        PARTITION BY name
        ORDER BY (namespace = 'default') IS TRUE DESC, updated_at DESC, created_at DESC, id DESC
      ) AS row_number
    FROM scopes
    WHERE deleted_at IS NULL AND (namespace IS NULL OR namespace IN ('', 'default'))
  )
  SELECT COALESCE(array_agg(id), '{}') INTO duplicates
  FROM ranked
  WHERE (namespace IS NULL OR namespace = '') AND row_number > 1;

  UPDATE scopes SET deleted_at = NOW() WHERE id = ANY(duplicates);

  IF to_regclass('public.scope_targets') IS NOT NULL THEN
    DELETE FROM scope_targets WHERE scope_id = ANY(duplicates);
  END IF;
  IF to_regclass('public.scope_members') IS NOT NULL THEN
    DELETE FROM scope_members WHERE scope_id = ANY(duplicates);
  END IF;

  UPDATE scopes SET namespace = 'default' WHERE namespace IS NULL OR namespace = '';
END $$;
