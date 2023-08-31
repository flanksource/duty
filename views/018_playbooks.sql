-- Notify playbook run created or status updated
CREATE OR REPLACE FUNCTION notify_playbook_run_update() RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    NOTIFY playbook_run_updates;
  ELSEIF TG_OP = 'UPDATE' THEN
    IF OLD.status != NEW.status AND NEW.status = 'scheduled' THEN
      NOTIFY playbook_run_updates;
    END IF;
  END IF;
    
  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_run_updates
AFTER INSERT OR UPDATE ON playbook_runs
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_run_update();

-- Notify playbook updates
CREATE OR REPLACE FUNCTION notify_playbook_update() 
RETURNS TRIGGER AS $$
DECLARE payload TEXT;
BEGIN
  payload = NEW.id::TEXT;
  PERFORM pg_notify('playbook_updated', payload);
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_updated_trigger
AFTER UPDATE ON playbooks
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_update();