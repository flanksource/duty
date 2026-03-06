-- Enforce element-level uniqueness for selected text[] columns using
-- deferrable constraint triggers + advisory locks.
--
-- Why this exists:
-- - UNIQUE on text[] only checks whole-array equality.
-- - PostgreSQL cannot do EXCLUDE USING GIN for && overlap checks.

-- Drop and recreate triggers because constraint triggers do not support OR REPLACE.
DROP TRIGGER IF EXISTS external_users_aliases_element_unique ON external_users;
DROP TRIGGER IF EXISTS external_groups_aliases_element_unique ON external_groups;
DROP TRIGGER IF EXISTS external_roles_aliases_element_unique ON external_roles;
DROP TRIGGER IF EXISTS config_items_external_id_element_unique ON config_items;

CREATE OR REPLACE FUNCTION check_array_no_overlap()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
  col_name    text := TG_ARGV[0];
  new_val     text[];
  elem        text;
  conflict_id uuid;
BEGIN
  EXECUTE format('SELECT ($1).%I', col_name) INTO new_val USING NEW;

  -- Only enforce for active rows with non-empty arrays.
  IF NEW.deleted_at IS NOT NULL OR COALESCE(cardinality(new_val), 0) = 0 THEN
    RETURN NEW;
  END IF;

  -- Serialize competing writers for each element to avoid races.
  -- Sorted order prevents lock-order deadlocks.
  FOR elem IN
    SELECT DISTINCT e
    FROM unnest(new_val) AS e
    WHERE e IS NOT NULL
    ORDER BY e
  LOOP
    PERFORM pg_advisory_xact_lock(
      hashtextextended(
        format('%I.%I.%I:%s', TG_TABLE_SCHEMA, TG_TABLE_NAME, col_name, elem),
        0
      )
    );
  END LOOP;

  -- Check if any active sibling row overlaps with the incoming array.
  EXECUTE format(
    'SELECT id
       FROM %I.%I
      WHERE deleted_at IS NULL
        AND id <> $1
        AND %I && $2
      LIMIT 1',
    TG_TABLE_SCHEMA, TG_TABLE_NAME, col_name
  )
  INTO conflict_id
  USING NEW.id, new_val;

  IF conflict_id IS NOT NULL THEN
    RAISE EXCEPTION 'conflicting array element in column "%" of table "%"', col_name, TG_TABLE_NAME
      USING ERRCODE = '23505',
            DETAIL = format('id=%s conflicts with existing id=%s', NEW.id, conflict_id);
  END IF;

  RETURN NEW;
END;
$$;

-- external_users.aliases element-level uniqueness
CREATE CONSTRAINT TRIGGER external_users_aliases_element_unique
AFTER INSERT OR UPDATE OF aliases, deleted_at ON external_users
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION check_array_no_overlap('aliases');

-- external_groups.aliases element-level uniqueness
CREATE CONSTRAINT TRIGGER external_groups_aliases_element_unique
AFTER INSERT OR UPDATE OF aliases, deleted_at ON external_groups
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION check_array_no_overlap('aliases');

-- external_roles.aliases element-level uniqueness
CREATE CONSTRAINT TRIGGER external_roles_aliases_element_unique
AFTER INSERT OR UPDATE OF aliases, deleted_at ON external_roles
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION check_array_no_overlap('aliases');

-- config_items.external_id element-level uniqueness
CREATE CONSTRAINT TRIGGER config_items_external_id_element_unique
AFTER INSERT OR UPDATE OF external_id, deleted_at ON config_items
DEFERRABLE INITIALLY IMMEDIATE
FOR EACH ROW
EXECUTE FUNCTION check_array_no_overlap('external_id');
