-- dependsOn: functions/drop.sql, views/006_config_views.sql
-- Handle before updates for notifications
CREATE OR REPLACE FUNCTION reset_notification_error_before_update ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF OLD.filter != NEW.filter OR OLD.custom_services != NEW.custom_services OR OLD.team_id != NEW.team_id THEN
    NEW.error = NULL;
  END IF;
  RETURN NEW;
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER reset_notification_error_before_update_trigger
  BEFORE UPDATE ON notifications
  FOR EACH ROW
  EXECUTE PROCEDURE reset_notification_error_before_update ();

--
CREATE OR REPLACE FUNCTION reset_notification_silence_error_before_update ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF OLD.filter != NEW.filter THEN
    NEW.error = NULL;
  END IF;
  RETURN NEW;
END
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER reset_notification_silence_error_before_update_trigger
  BEFORE UPDATE ON notification_silences
  FOR EACH ROW
  EXECUTE PROCEDURE reset_notification_silence_error_before_update ();

-- Ensure the previous function is cleaned up.
DROP FUNCTION IF EXISTS insert_unsent_notification_to_history(uuid, text, uuid, text, interval);
DROP FUNCTION IF EXISTS insert_unsent_notification_to_history(uuid, text, uuid, text, interval, uuid, uuid, uuid, uuid, uuid, uuid, text);

--- A function to insert only those notifications that were unsent.
--- It deals with the deduplication of inserting the same notification again if it was silenced or blocked by repeatInterval.
CREATE OR REPLACE FUNCTION insert_unsent_notification_to_history (
  p_notification_id uuid, 
  p_source_event text,
  p_resource_id uuid, 
  p_status text, 
  p_window interval,
  p_silenced_by uuid DEFAULT NULL,
  p_parent_id uuid DEFAULT NULL,
  p_person_id uuid DEFAULT NULL,
  p_team_id uuid DEFAULT NULL,
  p_connection_id uuid DEFAULT NULL,
  p_playbook_run_id uuid DEFAULT NULL,
  p_body text DEFAULT NULL,
  p_body_payload jsonb DEFAULT NULL
)
  RETURNS VOID
  AS $$
DECLARE
  v_existing_id uuid;
BEGIN
  IF p_status NOT IN ('silenced', 'inhibited', 'repeat-interval') THEN
    RAISE EXCEPTION 'Status must be silenced, inhibited or repeat-interval';
  END IF;

  SELECT
    id INTO v_existing_id
  FROM
    notification_send_history
  WHERE
    notification_id = p_notification_id
    AND source_event = p_source_event
    AND resource_id = p_resource_id
    AND status = p_status
    AND created_at > NOW() - p_window
    AND (p_status != 'inhibited' OR parent_id = p_parent_id)
  ORDER BY
    created_at DESC
  LIMIT 1;

  IF v_existing_id IS NOT NULL THEN
    UPDATE
      notification_send_history
    SET
      count = count + 1,
      body = p_body,
      body_payload = p_body_payload,
      created_at = CURRENT_TIMESTAMP
    WHERE
      id = v_existing_id;
  ELSE
    INSERT INTO notification_send_history (notification_id, status, source_event, resource_id, parent_id, silenced_by, person_id, team_id, connection_id, playbook_run_id, body, body_payload)
      VALUES (p_notification_id, p_status, p_source_event, p_resource_id, p_parent_id, p_silenced_by, p_person_id, p_team_id, p_connection_id, p_playbook_run_id, p_body, p_body_payload);
  END IF;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION skip_notification_send_history(
  p_send_history_id uuid, -- the original send history id that is to be skipped
  p_window interval
)
  RETURNS VOID
  AS $$
DECLARE
  v_existing_skipped uuid;
  v_existing_send_history record;
BEGIN
  -- Get the existing send history
  SELECT * INTO v_existing_send_history
  FROM notification_send_history
  WHERE id = p_send_history_id;

  IF v_existing_send_history.status NOT IN ('pending', 'evaluating-waitfor') THEN
    RAISE EXCEPTION 'status must be pending or evaluating-waitfor';
  END IF;

  -- Check if there is an existing send history with the status 'skipped'
  SELECT
    id INTO v_existing_skipped
  FROM
    notification_send_history
  WHERE
    notification_id = v_existing_send_history.notification_id
    AND source_event = v_existing_send_history.source_event
    AND resource_id = v_existing_send_history.resource_id
    AND status = 'skipped'
    AND created_at > NOW() - p_window
  ORDER BY
    created_at DESC
  LIMIT 1;

  IF v_existing_skipped IS NOT NULL THEN
    UPDATE
      notification_send_history
    SET
      count = count + 1,
      created_at = CURRENT_TIMESTAMP
    WHERE
      id = v_existing_skipped;

    -- Delete the old notification send history
    DELETE FROM notification_send_history
    WHERE id = p_send_history_id;
  ELSE
    UPDATE notification_send_history
    SET status = 'skipped'
    WHERE id = p_send_history_id;
  END IF;
END;
$$
LANGUAGE plpgsql;

---
DROP VIEW IF EXISTS notification_send_history_resource_tags;
DROP VIEW IF EXISTS notification_send_history_resource_types;
DROP VIEW IF EXISTS notification_send_history_summary;
DROP VIEW IF EXISTS notification_send_history_resources;

CREATE OR REPLACE VIEW notification_send_history_resources AS
SELECT 
  config_items.id,
  'config' AS "resource_kind",
  config_items.type AS "resource_type",
  jsonb_build_object(
    'id', config_items.id,
    'name', config_items.name,
    'type', config_items.type,
    'config_class', config_items.config_class,
    'health', config_items.health,
    'status', config_items.status,
    'deleted_at', config_items.deleted_at
  ) AS "resource"
FROM config_items JOIN notification_send_history 
ON config_items.id = notification_send_history.resource_id AND notification_send_history.source_event LIKE 'config.%'
UNION
SELECT
  components.id,
  'component' AS "resource_kind",
  components.type AS "resource_type",
  jsonb_build_object(
    'id', components.id,
    'name', components.name,
    'type', components.type,
    'icon', components.icon,
    'health', components.health,
    'status', components.status,
    'deleted_at', components.deleted_at
  ) AS "resource"
FROM components JOIN notification_send_history 
ON components.id = notification_send_history.resource_id AND notification_send_history.source_event LIKE 'component.%'
UNION
SELECT
  checks.id,
  'check' AS "resource_kind",
  checks.type AS "resource_type",
  jsonb_build_object(
    'id', checks.id,
    'name', checks.name,
    'type', checks.type,
    'icon', checks.icon,
    'health', checks.status,
    'status', checks.status,
    'deleted_at', checks.deleted_at
  ) AS "resource"
FROM checks JOIN notification_send_history 
ON checks.id = notification_send_history.resource_id AND notification_send_history.source_event LIKE 'check.%'
UNION
SELECT
  canaries.id,
  'canary' AS "resource_kind",
  NULL::text AS "resource_type",
  jsonb_build_object(
    'id', canaries.id,
    'name', canaries.name,
    'deleted_at', canaries.deleted_at
  ) AS "resource"
FROM canaries JOIN notification_send_history 
ON canaries.id = notification_send_history.resource_id AND notification_send_history.source_event LIKE 'canary.%';

--- notification_send_history_summary
-- NOTE: notification_send_history.body is legacy and empty for new notifications.
-- Use body_payload for the source of truth.
CREATE OR REPLACE VIEW notification_send_history_summary as
SELECT 
  notification_send_history.*,
  config_items.tags AS "resource_tags",
  "nsh_resources"."resource",
  "nsh_resources"."resource_kind",
  "nsh_resources"."resource_type",
  CASE
    WHEN notification_send_history.playbook_run_id IS NOT NULL THEN (
      SELECT jsonb_build_object(
        'id', pr.id::text,
        'playbook_id', pr.playbook_id::text,
        'status', pr.status,
        'playbook_name', COALESCE(p.title, p.name)
      )
      FROM playbook_runs pr
      JOIN playbooks p ON p.id = pr.playbook_id
      WHERE pr.id = notification_send_history.playbook_run_id
    )
    ELSE NULL
  END::jsonb AS playbook_run
FROM notification_send_history
LEFT JOIN notification_send_history_resources AS "nsh_resources" ON notification_send_history.resource_id = nsh_resources.id
LEFT JOIN config_items ON nsh_resources.resource_kind = 'config' AND config_items.id = notification_send_history.resource_id;

COMMENT ON COLUMN notification_send_history.body IS 'Deprecated: legacy rendered body; new notifications leave this field empty. Use body_payload instead.';

--- notification_send_history_resource_tags
CREATE OR REPLACE VIEW notification_send_history_resource_tags AS
WITH resource_tag_sets AS (
  SELECT DISTINCT resource_tags
  FROM notification_send_history_summary
  WHERE resource_tags IS NOT NULL
)
SELECT d.key, d.value
FROM resource_tag_sets
JOIN json_each_text(resource_tag_sets.resource_tags::json) d ON TRUE
GROUP BY d.key, d.value
ORDER BY d.key, d.value;

--- notification_send_history_resource_types
CREATE OR REPLACE VIEW notification_send_history_resource_types AS
SELECT DISTINCT
  resource_type
FROM notification_send_history_summary
WHERE resource_type IS NOT NULL
ORDER BY resource_type;

-- Insert notification_send_history updates as config_changes
CREATE OR REPLACE FUNCTION insert_notification_history_config_change()
RETURNS TRIGGER AS $$
DECLARE
    severity TEXT := 'info';
    change_type TEXT;
BEGIN

    -- All other status changes can be ignored
    IF NOT (NEW.status IN ('sent', 'attempting_fallback', 'error', 'silenced', 'inhibited', 'grouped', 'repeat-interval')) THEN
        RETURN NEW;
    END IF;

    -- Set severity based on status
    severity := CASE NEW.status
        WHEN 'error'    THEN 'high'
        WHEN 'attempting_fallback' THEN 'medium'
        ELSE 'info'
    END;

    -- Only config item notifications need to be inserted
    IF NEW.source_event LIKE 'config.%' AND ((TG_OP = 'INSERT') OR (TG_OP = 'UPDATE' AND OLD.status != NEW.status)) AND NEW.status != '' THEN
        INSERT INTO config_changes (config_id, change_type, source, details, external_change_id, severity)
        VALUES (NEW.resource_id, CONCAT('Notification', INITCAP(NEW.status)), 'notification', NEW.payload, CONCAT(NEW.id, '-', NEW.status, '-', CURRENT_TIMESTAMP), severity);
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER insert_notification_history_config_change_trigger
AFTER INSERT OR UPDATE ON notification_send_history
FOR EACH ROW
EXECUTE FUNCTION insert_notification_history_config_change();

---
CREATE OR REPLACE FUNCTION notification_send_history_of_resource(p_resource_id uuid)
RETURNS SETOF notification_send_history AS $$
BEGIN
  RETURN QUERY
  SELECT * FROM notification_send_history
  WHERE id IN (
    SELECT id FROM notification_send_history WHERE resource_id = p_resource_id
    UNION
    SELECT parent_id FROM notification_send_history WHERE resource_id = p_resource_id
  );
END;
$$ LANGUAGE plpgsql;
