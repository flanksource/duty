-- Cleanup previous trigger and function
DROP TRIGGER IF EXISTS handle_notifications_updates_deletes_trigger ON notifications;

DROP FUNCTION IF EXISTS handle_notifications_updates_deletes;

-- Notify on any updates/deletes on these tables
CREATE OR REPLACE FUNCTION notify_table_updates_and_deletes ()
  RETURNS TRIGGER
  AS $$
DECLARE
  payload text;
  record_id text;
BEGIN
  IF TG_TABLE_NAME = 'team_members' THEN
    IF TG_OP = 'DELETE' THEN
      record_id := OLD.team_id::text || ' ' || OLD.person_id;
    ELSE
      record_id := NEW.team_id::text || ' ' || NEW.person_id;
    END IF;
  ELSE
    IF TG_OP = 'DELETE' THEN
      record_id := OLD.id::text;
    ELSE
      record_id := NEW.id::text;
    END IF;
  END IF;

  payload := format('%s %s %s', TG_TABLE_NAME, TG_OP, record_id);
  PERFORM pg_notify('table_activity', payload);
  RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DO $$
DECLARE
  table_name text;
BEGIN
  FOR table_name IN
  SELECT
    unnest(ARRAY['notifications', 'playbooks', 'permissions', 'permission_groups', 'scopes', 'scrape_plugins', 'teams', 'team_members'])
    LOOP
      EXECUTE format('
      CREATE OR REPLACE TRIGGER notify_updates_and_deletes
      AFTER INSERT OR UPDATE OR DELETE ON %I
      FOR EACH ROW
      EXECUTE PROCEDURE notify_table_updates_and_deletes()', table_name);
    END LOOP;
END
$$;

---
CREATE OR REPLACE FUNCTION notify_completed_playbook_actions ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF NEW.agent_id IS NULL THEN
    RETURN NULL;
  END IF;
  IF OLD.end_time IS NULL AND NEW.end_time IS NOT NULL THEN
    PERFORM
      pg_notify('table_activity', TG_TABLE_NAME || ' ' || NEW.id);
  END IF;
  RETURN NULL;
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER newly_completed_actions
  AFTER UPDATE ON playbook_run_actions
  FOR EACH ROW
  EXECUTE PROCEDURE notify_completed_playbook_actions ();

