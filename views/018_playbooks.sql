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

-- List of all the playbooks that can be run by an agent
CREATE OR REPLACE VIEW 
playbooks_for_agent AS
WITH interim AS (
  SELECT
    id,
    jsonb_array_elements_text(spec -> 'runsOn') AS agent_name
  FROM
    playbooks
  WHERE
    spec ->> 'runsOn' IS NOT NULL
)
SELECT
  interim.agent_name,
  agents.person_id,
  agents.id as agent_id,
  json_agg(interim.id) AS playbook_ids
FROM
  interim
  INNER JOIN agents ON interim.agent_name :: TEXT = agents.name
GROUP BY agent_name, agents.person_id, agent_id