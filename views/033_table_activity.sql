-- Cleanup previous trigger and function
DROP TRIGGER IF EXISTS handle_notifications_updates_deletes_trigger ON notifications;

DROP FUNCTION IF EXISTS handle_notifications_updates_deletes;

-- Notify on any updates/deletes on these tables
CREATE OR REPLACE FUNCTION notify_table_updates_and_deletes ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    PERFORM
      pg_notify('table_activity', TG_TABLE_NAME || ' ' || OLD.id);
  ELSE
    PERFORM
      pg_notify('table_activity', TG_TABLE_NAME || ' ' || NEW.id);
  END IF;
  RETURN NULL;
END
$$
LANGUAGE plpgsql;

DO $$
DECLARE
  table_name text;
BEGIN
  FOR table_name IN
  SELECT
    unnest(ARRAY['notifications', 'playbooks', 'permissions', 'scrape_plugins', 'teams'])
    LOOP
      EXECUTE format('
      CREATE OR REPLACE TRIGGER notify_updates_and_deletes
      AFTER INSERT OR UPDATE OR DELETE ON %I
      FOR EACH ROW
      EXECUTE PROCEDURE notify_table_updates_and_deletes()', table_name);
    END LOOP;
END
$$;

