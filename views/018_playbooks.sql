-- Notify playbook run created
CREATE OR REPLACE FUNCTION notify_playbook_run_insertion() RETURNS TRIGGER AS $$
BEGIN
  NOTIFY playbook_run_updates;
  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_run_insertion
AFTER INSERT ON playbook_runs
FOR EACH ROW
EXECUTE PROCEDURE notify_playbook_run_insertion();