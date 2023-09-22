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

CREATE OR REPLACE TRIGGER handle_notifications_updates_deletes_trigger
AFTER UPDATE OR DELETE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE handle_notifications_updates_deletes();

-- Handle before updates for notifications
CREATE OR REPLACE FUNCTION reset_notification_error_before_update()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.filter != NEW.filter OR OLD.custom_services != NEW.custom_services OR OLD.team_id != NEW.team_id THEN
    NEW.error = NULL;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER reset_notification_error_before_update_trigger
BEFORE UPDATE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE reset_notification_error_before_update();