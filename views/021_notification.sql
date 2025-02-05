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

--- A function to insert only those notifications that were unsent.
--- It deals with the deduplication of inserting the same notification again if it was silenced or blocked by repeatInterval.
CREATE OR REPLACE FUNCTION insert_unsent_notification_to_history (
  p_notification_id uuid, 
  p_source_event text,
  p_resource_id uuid, 
  p_status text, 
  p_window interval,
  p_silenced_by uuid DEFAULT NULL
)
  RETURNS VOID
  AS $$
DECLARE
  v_existing_id uuid;
BEGIN
  IF p_status NOT IN ('silenced', 'repeat-interval') THEN
    RAISE EXCEPTION 'Status must be silenced or repeat-interval';
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
  ORDER BY
    created_at DESC
  LIMIT 1;

  IF v_existing_id IS NOT NULL THEN
    UPDATE
      notification_send_history
    SET
      count = count + 1,
      created_at = CURRENT_TIMESTAMP
    WHERE
      id = v_existing_id;
  ELSE
    INSERT INTO notification_send_history (notification_id, status, source_event, resource_id, silenced_by)
      VALUES (p_notification_id, p_status, p_source_event, p_resource_id, p_silenced_by);
  END IF;
END;
$$
LANGUAGE plpgsql;

DROP VIEW IF EXISTS notification_send_history_summary;

CREATE OR REPLACE VIEW notification_send_history_summary AS
WITH combined AS (
  -- config
  SELECT
    nsh.*,
    'config' AS "resource_type",
    jsonb_build_object('id', config.id, 'name', config.name, 'type', config.type, 'config_class', config.config_class) AS resource
  FROM
    notification_send_history nsh
    LEFT JOIN (
      SELECT
        id,
        name,
        type,
        config_class
      FROM
        configs AS configs) config ON config.id = nsh.resource_id
    WHERE
      nsh.source_event LIKE 'config.%'
    UNION
    -- component
    SELECT
      nsh.*,
      'component' AS "resource_type",
      jsonb_build_object('id', component.id, 'name', component.name, 'icon', component.icon) AS resource
    FROM
      notification_send_history nsh
    LEFT JOIN (
      SELECT
        id,
        name,
        icon
      FROM
        components) component ON component.id = nsh.resource_id
    WHERE
      nsh.source_event LIKE 'component.%'
    UNION
    -- check
    SELECT
      nsh.*,
      'check' AS "resource_type",
      jsonb_build_object('id', check_details.id, 'name', check_details.name, 'type', check_details.type, 'status', check_details.status, 'icon', check_details.icon) AS resource
    FROM
      notification_send_history nsh
    LEFT JOIN (
      SELECT
        id,
        name,
        type,
        status,
        icon
      FROM
        checks) check_details ON check_details.id = nsh.resource_id
    WHERE
      nsh.source_event LIKE 'check.%'
    UNION
    -- canary
    SELECT
      nsh.*,
      'canary' AS "resource_type",
      jsonb_build_object('id', canary.id, 'name', canary.name) AS resource
    FROM
      notification_send_history nsh
    LEFT JOIN (
      SELECT
        id,
        name
      FROM
        canaries) canary ON canary.id = nsh.resource_id
    WHERE
      nsh.source_event LIKE 'canary.%'
)
SELECT
  combined.*
FROM
  combined;


-- Insert notification_send_history updates as config_changes
CREATE OR REPLACE FUNCTION insert_notification_history_config_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only config item notifications need to be inserted
    IF NEW.source_event LIKE 'config.%' AND ((TG_OP = 'INSERT') OR (TG_OP = 'UPDATE' AND OLD.status != NEW.status)) AND NEW.status != '' THEN
        INSERT INTO config_changes (config_id, change_type, source, details, external_change_id)
        VALUES (NEW.resource_id, CONCAT('Notification', INITCAP(NEW.status)), 'notification', NEW.payload, CONCAT(NEW.id, '-', NOW()));
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER insert_notification_history_config_change_trigger
AFTER INSERT OR UPDATE ON notification_send_history
FOR EACH ROW
EXECUTE FUNCTION insert_notification_history_config_change();
