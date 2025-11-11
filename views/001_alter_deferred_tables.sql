-- Function to check if a constraint is deferrable
CREATE OR REPLACE FUNCTION constraint_is_deferrable(p_table_name TEXT, p_constraint_name TEXT)
  RETURNS BOOLEAN AS $$
  DECLARE
    v_is_deferrable BOOLEAN;
  BEGIN
    SELECT c.condeferrable INTO v_is_deferrable
    FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE t.relname = p_table_name
    AND c.conname = p_constraint_name;
    RETURN v_is_deferrable;
  END;
$$ LANGUAGE plpgsql;

-- Update constraints to be DEFERRABLE only if not already deferrable
DO $$
BEGIN
  IF NOT constraint_is_deferrable('config_analysis', 'config_analysis_config_id_fkey') THEN
    ALTER TABLE config_analysis
    ALTER CONSTRAINT config_analysis_config_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
    IF NOT constraint_is_deferrable('config_items', 'config_items_parent_id_fkey') THEN
    ALTER TABLE config_items
    ALTER CONSTRAINT config_items_parent_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
  IF NOT constraint_is_deferrable('config_changes', 'config_changes_config_id_fkey') THEN
    ALTER TABLE config_changes
    ALTER CONSTRAINT config_changes_config_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
  IF NOT constraint_is_deferrable('config_relationships', 'config_relationships_config_id_fkey') THEN
    ALTER TABLE config_relationships
    ALTER CONSTRAINT config_relationships_config_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
  IF NOT constraint_is_deferrable('config_relationships', 'config_relationships_related_id_fkey') THEN
    ALTER TABLE config_relationships
    ALTER CONSTRAINT config_relationships_related_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
  IF NOT constraint_is_deferrable('check_config_relationships', 'check_config_relationships_config_id_fkey') THEN
    ALTER TABLE check_config_relationships
    ALTER CONSTRAINT check_config_relationships_config_id_fkey DEFERRABLE INITIALLY DEFERRED;
  END IF;
END;
$$;

CREATE OR REPLACE PROCEDURE delete_old_config_items(older_than_days INT)
LANGUAGE plpgsql
AS $$
DECLARE
  v_batch_size INT := 1000;
  v_deleted_count INT;
  v_total_deleted INT := 0;
BEGIN
  -- Create a temporary table to store config_item IDs to ignore
  DROP TABLE IF EXISTS ignored_config_items;
  CREATE TEMP TABLE ignored_config_items AS
    SELECT DISTINCT config_id FROM evidences
    UNION SELECT DISTINCT config_id FROM playbook_runs
    UNION SELECT DISTINCT config_id FROM components;

  LOOP
    -- Create a temporary table to store config_item IDs to delete for each batch
    DROP TABLE IF EXISTS config_items_to_delete;

    CREATE TEMP TABLE config_items_to_delete AS
      SELECT id
      FROM config_items
      WHERE deleted_at < NOW() - INTERVAL '1 day' * older_than_days
      AND NOT EXISTS (
        SELECT 1
        FROM ignored_config_items ici
        WHERE ici.config_id = config_items.id
      )
      LIMIT v_batch_size;

    -- Check if we have any items to delete
    SELECT COUNT(*) INTO v_deleted_count FROM config_items_to_delete;

    EXIT WHEN v_deleted_count = 0;

    -- Set constraints deferred for this batch
    SET CONSTRAINTS ALL DEFERRED;

    -- Delete related data in batches using the config_items_to_delete table
    DELETE FROM config_analysis
    WHERE config_id IN (SELECT id FROM config_items_to_delete);

    DELETE FROM config_changes
    WHERE config_id IN (SELECT id FROM config_items_to_delete);

    DELETE FROM config_relationships
    WHERE config_id IN (SELECT id FROM config_items_to_delete)
    OR related_id IN (SELECT id FROM config_items_to_delete);

    DELETE FROM check_config_relationships
    WHERE config_id IN (SELECT id FROM config_items_to_delete);

    -- Components are independent so we just unset config_id
    UPDATE components SET config_id = NULL
    WHERE config_id IN (SELECT id FROM config_items_to_delete);

    -- Update parent_id references before deleting
    UPDATE config_items SET parent_id = NULL WHERE parent_id IN (SELECT id FROM config_items_to_delete);

    -- Finally, delete the config_items themselves
    DELETE FROM config_items
    WHERE id IN (SELECT id FROM config_items_to_delete);

    v_total_deleted := v_total_deleted + v_deleted_count;

    -- Drop the temporary table for this batch
    DROP TABLE config_items_to_delete;

    -- Commit this batch to persist the changes
    COMMIT;

  END LOOP;

  DROP TABLE IF EXISTS ignored_config_items;

  RAISE NOTICE 'Total config items deleted: %', v_total_deleted;
END;
$$;
