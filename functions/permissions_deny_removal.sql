-- Deny rules are removed. Runs ahead of the schema apply, which no longer has permissions.deny.
--
-- Every Permission stored with deny set is deleted in the same transaction that drops the column:
-- without the column, each of them would read as an allow of the same subject, object and action.
-- A Role with a deny rule is deleted too, for the same reason once its rules are read without deny,
-- and so is a stored Casbin deny policy, which no longer has any effect.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'permissions' AND column_name = 'deny'
  ) THEN
    DROP FUNCTION IF EXISTS permissions_for_obj_selector(TEXT, TEXT, TEXT);
    DROP VIEW IF EXISTS permissions_summary;
    DROP VIEW IF EXISTS permissions_group_summary;

    DELETE FROM permissions WHERE deny;
    ALTER TABLE permissions DROP COLUMN deny;
  END IF;

  IF to_regclass('public.roles') IS NOT NULL THEN
    DELETE FROM roles
    WHERE jsonb_typeof(rules) = 'array'
      AND EXISTS (
        SELECT 1 FROM jsonb_array_elements(rules) AS rule
        WHERE jsonb_typeof(rule) = 'object' AND rule->'deny' = 'true'::jsonb
      );
  END IF;

  IF to_regclass('public.casbin_rule') IS NOT NULL THEN
    DELETE FROM casbin_rule WHERE ptype = 'p' AND v3 = 'deny';
  END IF;
END $$;
