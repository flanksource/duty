-- Insert responder updates in event_queue
CREATE
OR REPLACE FUNCTION insert_responder_in_event_queue () RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO event_queue(name, properties) VALUES ('responder.create', jsonb_build_object('type', 'responder', 'id', NEW.id));
    NOTIFY event_queue_updates, 'update';
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER responder_enqueue
AFTER INSERT ON responders FOR EACH ROW
EXECUTE PROCEDURE insert_responder_in_event_queue ();

-- Insert comment updates in event_queue
CREATE
OR REPLACE FUNCTION insert_comment_in_event_queue () RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO event_queue(name, properties) VALUES ('comment.create', jsonb_build_object('type', 'comment', 'id', NEW.id, 'body', NEW.comment));
    NOTIFY event_queue_updates, 'update';
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER comment_enqueue
AFTER INSERT ON comments FOR EACH ROW
EXECUTE PROCEDURE insert_comment_in_event_queue ();

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
    INSERT INTO event_queue(name, properties) VALUES ('team.delete', jsonb_build_object('team_id', OLD.id));
    NOTIFY event_queue_updates, 'update';
    RETURN OLD;
  ELSE
    INSERT INTO event_queue(name, properties) VALUES ('team.update', jsonb_build_object('team_id', NEW.id));
    NOTIFY event_queue_updates, 'update';
    RETURN NEW;
  END IF;
END
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER team_enqueue
AFTER INSERT OR UPDATE OR DELETE ON teams FOR EACH ROW
EXECUTE PROCEDURE insert_team_in_event_queue ();
