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

CREATE OR REPLACE TRIGGER playbook_run_insertion
AFTER INSERT ON playbook_runs
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_run_insertion();

-- Notify playbook `spec.approval` updated
CREATE OR REPLACE FUNCTION notify_playbook_spec_approval_update() 
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.spec->'approval' != NEW.spec->'approval' THEN
    NOTIFY playbook_approval_updated;
  END IF;
    
  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_spec_approval_update
AFTER UPDATE ON playbooks
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_spec_approval_update();

-- Notify playbook approvals insertion
CREATE OR REPLACE FUNCTION notify_playbook_approvals_insert() 
RETURNS TRIGGER AS $$
BEGIN
  NOTIFY playbook_approval_updated;
  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_approvals_insert
AFTER INSERT ON playbook_approvals
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_approvals_insert();