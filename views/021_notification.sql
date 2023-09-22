-- Notify on any updates/deletes on the notifications table
CREATE OR REPLACE FUNCTION handle_notifications_updates_deletes()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    PERFORM pg_notify('table_activity', TG_TABLE_NAME || ' ' || OLD.id);
  ELSE
    PERFORM pg_notify('table_activity', TG_TABLE_NAME || ' ' || NEW.id);
  END IF;

  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER notification_update_enqueue
AFTER UPDATE OR DELETE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE handle_notifications_updates_deletes();