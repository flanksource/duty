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

    IF NEW.status != 'healthy' OR NEW.status != 'unhealthy' THEN
        RETURN NULL;
    END IF;

    IF NEW.status = 'healthy' THEN
        INSERT INTO event_queue(name, properties) VALUES ('check.passed', jsonb_build_object('id', NEW.id, 'last_runtime', NEW.last_runtime))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    ELSEIF NEW.status = 'unhealthy' THEN
        INSERT INTO event_queue(name, properties) VALUES ('check.failed', jsonb_build_object('id', NEW.id, 'last_runtime', NEW.last_runtime))
        ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    END IF;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_enqueue
AFTER UPDATE ON checks_unlogged
FOR EACH ROW
EXECUTE PROCEDURE insert_check_updates_in_event_queue ();

-- Insert config health updates in event_queue
CREATE OR REPLACE FUNCTION insert_config_health_updates_in_event_queue ()
RETURNS TRIGGER AS $$
DECLARE
    event_name TEXT;
BEGIN
    IF TG_OP = 'INSERT' THEN
      IF NEW.health NOT IN ('warning', 'unhealthy') THEN
        RETURN NUll;
      END IF;
    END IF;

    IF OLD.health = NEW.health OR (OLD.health IS NULL AND NEW.health IS NULL) THEN
      RETURN NULL;
    END IF;

    -- Special case: unhealthy to warning should emit degraded event
    IF OLD.health = 'unhealthy' AND NEW.health = 'warning' THEN
        event_name := 'config.degraded';
    ELSE
        event_name := CONCAT('config.', COALESCE(NULLIF(NEW.health, ''), 'unknown'));
    END IF;

    INSERT INTO event_queue(name, properties)
    VALUES (event_name, jsonb_build_object('id', NEW.id, 'status', NEW.status, 'description', NEW.description))
    ON CONFLICT (name, properties) DO UPDATE SET
        created_at = NOW(),
        last_attempt = NULL,
        attempts = 0;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_health_event_enqueue
AFTER INSERT OR UPDATE ON config_items
FOR EACH ROW
EXECUTE PROCEDURE insert_config_health_updates_in_event_queue();

-- Cleanup
DROP TRIGGER IF EXISTS  component_status_enqueue ON components;
DROP FUNCTION IF EXISTS insert_component_status_updates_in_event_queue;

-- Insert component health updates in event_queue
CREATE OR REPLACE FUNCTION insert_component_health_updates_in_event_queue()
RETURNS TRIGGER AS $$
DECLARE
    event_name TEXT;
BEGIN
    IF OLD.health = NEW.health OR (OLD.health IS NULL AND NEW.health IS NULL) THEN
      RETURN NULL;
    END IF;

    event_name := CONCAT('component.', COALESCE(NULLIF(NEW.health, ''), 'unknown'));

    INSERT INTO event_queue (name, properties) VALUES (event_name, jsonb_build_object('id', NEW.id, 'status', NEW.status, 'description', NEW.description))
    ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;

    RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER component_health_enqueue
AFTER UPDATE ON components
FOR EACH ROW
EXECUTE PROCEDURE insert_component_health_updates_in_event_queue();

DROP VIEW IF EXISTS push_queue_summary;

CREATE VIEW push_queue_summary AS
SELECT 'artifacts' AS table, (SELECT COUNT(*) FROM artifacts WHERE is_pushed = false) AS count UNION
SELECT 'canaries' AS table, (SELECT COUNT(*) FROM canaries WHERE is_pushed = false) AS count UNION
SELECT 'check_component_relationships' AS table, (SELECT COUNT(*) FROM check_component_relationships WHERE is_pushed = false) AS count UNION
SELECT 'check_config_relationships' AS table, (SELECT COUNT(*) FROM check_config_relationships WHERE is_pushed = false) AS count UNION
SELECT 'check_statuses' AS table, (SELECT COUNT(*) FROM check_statuses WHERE is_pushed = false) AS count UNION
SELECT 'checks' AS table, (SELECT COUNT(*) FROM checks WHERE is_pushed = false) AS count UNION
SELECT 'component_relationships' AS table, (SELECT COUNT(*) FROM component_relationships WHERE is_pushed = false) AS count UNION
SELECT 'components' AS table, (SELECT COUNT(*) FROM components WHERE is_pushed = false) AS count UNION
SELECT 'config_analysis' AS table, (SELECT COUNT(*) FROM config_analysis WHERE is_pushed = false) AS count UNION
SELECT 'config_changes' AS table, (SELECT COUNT(*) FROM config_changes WHERE is_pushed = false) AS count UNION
SELECT 'config_component_relationships' AS table, (SELECT COUNT(*) FROM config_component_relationships WHERE is_pushed = false) AS count UNION
SELECT 'config_items' AS table, (SELECT COUNT(*) FROM config_items WHERE is_pushed = false) AS count UNION
SELECT 'config_relationships' AS table, (SELECT COUNT(*) FROM config_relationships WHERE is_pushed = false) AS count UNION
SELECT 'config_scrapers' AS table, (SELECT COUNT(*) FROM config_scrapers WHERE is_pushed = false) AS count UNION
SELECT 'playbook_run_actions' AS table, (SELECT COUNT(*) FROM playbook_run_actions WHERE is_pushed = false) AS count UNION
SELECT 'topologies' AS table, (SELECT COUNT(*) FROM topologies WHERE is_pushed = false) AS count;

CREATE OR REPLACE VIEW event_queue_summary AS
SELECT
  name,
  COUNT(id) AS pending,
  COUNT(CASE WHEN error IS NOT NULL THEN 1 END) AS failed,
  ROUND(AVG(attempts)::numeric, 2) AS average_attempts,
  MIN(CASE WHEN error IS NOT NULL THEN created_at END) AS first_failure,
  MAX(last_attempt) AS last_failure,
  mode() WITHIN GROUP (ORDER BY error) AS most_common_error
FROM
  event_queue
GROUP BY
  name;

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
