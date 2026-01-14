-- Emits scope/permission materialization events into event_queue.
-- This keeps precomputed __scope columns in sync for both CRD and UI writes
-- by delegating all updates to the application event processor.
CREATE OR REPLACE FUNCTION insert_scope_materialization_event()
RETURNS TRIGGER AS $$
DECLARE
  action TEXT;
BEGIN
  IF TG_OP = 'INSERT' THEN
    action := 'rebuild';
  ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
      action := 'remove';
    ELSE
      action := 'rebuild';
    END IF;
  ELSE
    RETURN NEW;
  END IF;

  INSERT INTO event_queue(name, properties)
  VALUES ('scope.materialize', jsonb_build_object('id', NEW.id::text, 'action', action))
  ON CONFLICT (name, properties) DO UPDATE
    SET created_at = NOW(), last_attempt = NULL, attempts = 0;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER scopes_materialize_event_trigger
AFTER INSERT OR UPDATE ON scopes
FOR EACH ROW
EXECUTE FUNCTION insert_scope_materialization_event();

CREATE OR REPLACE FUNCTION insert_permission_materialization_event()
RETURNS TRIGGER AS $$
DECLARE
  action TEXT;
BEGIN
  IF TG_OP = 'INSERT' THEN
    action := 'rebuild';
  ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
      action := 'remove';
    ELSE
      action := 'rebuild';
    END IF;
  ELSE
    RETURN NEW;
  END IF;

  INSERT INTO event_queue(name, properties)
  VALUES ('permission.materialize', jsonb_build_object('id', NEW.id::text, 'action', action))
  ON CONFLICT (name, properties) DO UPDATE
    SET created_at = NOW(), last_attempt = NULL, attempts = 0;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER permissions_materialize_event_trigger
AFTER INSERT OR UPDATE ON permissions
FOR EACH ROW
EXECUTE FUNCTION insert_permission_materialization_event();
