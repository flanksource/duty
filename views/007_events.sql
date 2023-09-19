-- Insert playbook `spec.approval` updates to event queue
CREATE OR REPLACE FUNCTION insert_playbook_spec_approval_in_event_queue() 
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.spec->'approval' != NEW.spec->'approval' THEN
    INSERT INTO event_queue(name, properties) VALUES ('playbook.spec.approval.updated', jsonb_build_object('id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
  END IF;
    
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER enqueue_playbook_spec_approval_updated
AFTER UPDATE ON playbooks
FOR EACH ROW
EXECUTE PROCEDURE insert_playbook_spec_approval_in_event_queue();

-- Insert new playbook approvals to event queue
CREATE OR REPLACE FUNCTION insert_new_playbook_approvals_to_event_queue() 
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO event_queue(name, properties) VALUES ('playbook.approval.inserted', jsonb_build_object('id', NEW.id, 'run_id', NEW.run_id))
  ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER enqueue_new_playbook_approval
AFTER INSERT ON playbook_approvals
FOR EACH ROW
EXECUTE PROCEDURE insert_new_playbook_approvals_to_event_queue();

-- Insert incident created in event_queue
CREATE OR REPLACE FUNCTION insert_incident_creation_in_event_queue() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO event_queue(name, properties) VALUES ('incident.created', jsonb_build_object('id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER incident_enqueue
AFTER INSERT ON incidents
FOR EACH ROW
EXECUTE PROCEDURE insert_incident_creation_in_event_queue ();

-- INSERT incident status updates in event_queue
CREATE OR REPLACE FUNCTION insert_incident_updates_in_event_queue() RETURNS TRIGGER AS $$
DECLARE event_name TEXT;
BEGIN
    IF OLD.status = NEW.status THEN
        RETURN NULL;
    END IF;

    event_name := 'incident.status.' || NEW.status;
    INSERT INTO event_queue(name, properties) VALUES (event_name, jsonb_build_object('id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER incident_status_enque
AFTER UPDATE ON incidents
FOR EACH ROW
EXECUTE PROCEDURE insert_incident_updates_in_event_queue();

-- Insert incident responder updates in event_queue
CREATE OR REPLACE FUNCTION insert_responder_in_event_queue() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO event_queue(name, properties) VALUES ('incident.responder.added', jsonb_build_object('id', NEW.id))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            INSERT INTO event_queue(name, properties) VALUES ('incident.responder.removed', jsonb_build_object('id', NEW.id))
            ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
        END IF;
    END IF;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER responder_enqueue
AFTER INSERT OR UPDATE ON responders
FOR EACH ROW
EXECUTE PROCEDURE insert_responder_in_event_queue();

-- Insert incident comment creation in event_queue
CREATE OR REPLACE FUNCTION insert_comment_in_event_queue () RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO event_queue(name, properties) VALUES ('incident.comment.added', jsonb_build_object('id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER comment_enqueue
AFTER INSERT ON comments
FOR EACH ROW
EXECUTE PROCEDURE insert_comment_in_event_queue ();

-- Insert definition of done updates
CREATE OR REPLACE FUNCTION insert_definition_of_done_updates_in_event_queue() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.definition_of_done = NEW.definition_of_done AND OLD.done = NEW.done THEN
        RETURN NULL;
    END IF;

    IF OLD.definition_of_done != NEW.definition_of_done THEN
        IF NEW.definition_of_done THEN
            INSERT INTO event_queue(name, properties) VALUES ('incident.dod.added', jsonb_build_object('id', NEW.id))
            ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
        ELSE
            INSERT INTO event_queue(name, properties) VALUES ('incident.dod.removed', jsonb_build_object('id', NEW.id))
            ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
        END IF;
    END IF;

    IF OLD.done != NEW.done THEN
        IF NEW.done THEN
            INSERT INTO event_queue(name, properties) VALUES ('incident.dod.passed', jsonb_build_object('id', NEW.id))
            ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
        ELSE
            INSERT INTO event_queue(name, properties) VALUES ('incident.dod.regressed', jsonb_build_object('id', NEW.id))
            ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
        END IF;
    END IF;
    
    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER evidence_dod_updates
AFTER UPDATE ON evidences 
FOR EACH ROW
EXECUTE PROCEDURE insert_definition_of_done_updates_in_event_queue();

-- Insert check status updates in event_queue
CREATE OR REPLACE FUNCTION insert_check_updates_in_event_queue () RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status = NEW.status THEN
      RETURN NULL;
    END IF;

    IF NEW.status = 'healthy' THEN
        INSERT INTO event_queue(name, properties) VALUES ('check.passed', jsonb_build_object('id', NEW.id))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    ELSEIF NEW.status = 'unhealthy' THEN
        INSERT INTO event_queue(name, properties) VALUES ('check.failed', jsonb_build_object('id', NEW.id))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    END IF;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_enqueue
AFTER UPDATE ON checks
FOR EACH ROW
EXECUTE PROCEDURE insert_check_updates_in_event_queue ();

-- Insert component status updates in event_queue
CREATE OR REPLACE FUNCTION insert_component_status_updates_in_event_queue() RETURNS TRIGGER AS $$
DECLARE event_name TEXT;
BEGIN
    IF OLD.status = NEW.status THEN
      RETURN NULL;
    END IF;

    IF NEW.status = 'unknown' THEN
      RETURN NULL;
    END If;

    event_name := 'component.status.' || NEW.status;

    INSERT INTO event_queue (name, properties) VALUES (event_name, jsonb_build_object('id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER component_status_enqueue
AFTER UPDATE ON components
FOR EACH ROW
EXECUTE PROCEDURE insert_component_status_updates_in_event_queue();

CREATE OR REPLACE VIEW failed_push_queue AS
SELECT
  properties ->> 'table' AS "table",
  COUNT(id) AS error_count,
  ROUND(AVG(attempts)::numeric, 2) AS average_attempts,
  MIN(created_at) AS first_failure,
  MAX(last_attempt) AS latest_failure,
  mode() WITHIN GROUP (ORDER BY error) AS most_common_error
FROM
  event_queue
WHERE
  error IS NOT NULL AND attempts > 0
GROUP BY
  "table";

CREATE OR REPLACE VIEW failed_events AS
SELECT
  name,
  COUNT(DISTINCT error) AS unique_errors,
  ROUND(AVG(attempts)::numeric, 2) AS average_attempts,
  COUNT(*) AS total_failed_events,
  mode() WITHIN GROUP (ORDER BY error) AS most_common_error
FROM
  event_queue
WHERE
  error IS NOT NULL
  AND attempts > 0
  AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY
  name;

-- Insert team updates in event_queue
CREATE
OR REPLACE FUNCTION insert_team_in_event_queue () RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    INSERT INTO event_queue(name, properties) VALUES ('team.delete', jsonb_build_object('team_id', OLD.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
  ELSE
    INSERT INTO event_queue(name, properties) VALUES ('team.update', jsonb_build_object('team_id', NEW.id))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
  END IF;
  
  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER team_enqueue
AFTER INSERT OR UPDATE OR DELETE ON teams FOR EACH ROW
EXECUTE PROCEDURE insert_team_in_event_queue ();

-- Create new event on any updates on the notifications table
CREATE OR REPLACE FUNCTION notifications_trigger_function()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO event_queue(name, properties) VALUES ('notification.delete', jsonb_build_object('id', OLD.id))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    ELSE
        INSERT INTO event_queue(name, properties) VALUES ('notification.update', jsonb_build_object('id', NEW.id))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    END IF;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER notification_update_enqueue
AFTER INSERT OR UPDATE OR DELETE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE notifications_trigger_function ();

-- Publish Notify on new events
CREATE OR REPLACE FUNCTION notify_new_events_function()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM pg_notify('event_queue_updates', NEW.name);
        RETURN NULL;
    END IF;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER notify_new_events
AFTER INSERT ON event_queue
FOR EACH ROW
EXECUTE PROCEDURE notify_new_events_function();