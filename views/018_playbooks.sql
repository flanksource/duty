-- Notify playbook action created or status updated
CREATE OR REPLACE FUNCTION notify_playbook_action_update() RETURNS TRIGGER AS $$
  BEGIN
    IF TG_OP = 'INSERT' THEN
      PERFORM pg_notify('playbook_action_updates', json_build_object('id', NEW.id, 'agent_id', NEW.agent_id)::TEXT);
    ELSEIF TG_OP = 'UPDATE' THEN
      IF OLD.status != NEW.status AND NEW.status = 'scheduled' THEN
        PERFORM pg_notify('playbook_action_updates', json_build_object('id', NEW.id, 'agent_id', NEW.agent_id)::TEXT);
      END IF;
    END IF;

    RETURN NULL;
  END
  $$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER playbook_action_updates
  AFTER INSERT OR UPDATE ON playbook_run_actions
  FOR EACH ROW
  EXECUTE PROCEDURE notify_playbook_action_update();

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
DROP VIEW IF EXISTS playbooks_for_agent;

CREATE OR REPLACE VIEW playbooks_for_agent AS
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
    jsonb_agg(interim.id) AS playbook_ids
  FROM
    interim
    INNER JOIN agents ON interim.agent_name :: TEXT = agents.name
  GROUP BY agent_name, agents.person_id, agent_id;

DROP VIEW IF EXISTS playbook_names;
CREATE OR REPLACE VIEW playbook_names AS
  SELECT
    id,
    title,
    name,
    spec ->> 'description' AS description,
    spec ->> 'category' AS category,
    spec ->> 'icon' AS icon
  FROM
    playbooks
  WHERE
    deleted_at IS NULL
  ORDER BY
    name;

---
DROP FUNCTION IF EXISTS get_playbook_run_actions;

CREATE OR REPLACE FUNCTION get_playbook_run_actions(run_id uuid)
RETURNS TABLE (
    id uuid,
    name text,
    playbook_run_id uuid,
    status text,
    scheduled_time timestamp with time zone,
    start_time timestamp with time zone,
    end_time timestamp with time zone,
    agent_id uuid,
    retry_count integer,
    agent_name text
) AS $$
BEGIN
  RETURN QUERY
  WITH child_runs AS (
    SELECT child.id AS child_id FROM playbook_runs AS child WHERE child.parent_id = run_id
  )
  SELECT
    playbook_run_actions.id,
    playbook_run_actions.name,
    playbook_run_actions.playbook_run_id,
    playbook_run_actions.status,
    playbook_run_actions.scheduled_time,
    playbook_run_actions.start_time,
    playbook_run_actions.end_time,
    playbook_run_actions.agent_id,
    playbook_run_actions.retry_count,
    agents.name AS agent_name
  FROM playbook_run_actions
  LEFT JOIN agents ON playbook_run_actions.agent_id = agents.id
  WHERE playbook_run_actions.playbook_run_id IN (SELECT child_id FROM child_runs) OR playbook_run_actions.playbook_run_id = run_id
  ORDER BY playbook_run_actions.start_time;
END;
$$ LANGUAGE plpgsql;
