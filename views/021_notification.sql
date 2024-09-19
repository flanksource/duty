-- Notify on any updates/deletes on the notifications table
CREATE OR REPLACE FUNCTION handle_notifications_updates_deletes()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    PERFORM pg_notify('table_activity', TG_TABLE_NAME || ' ' || OLD.id);
  ELSE
    PERFORM pg_notify('table_activity', TG_TABLE_NAME || ' ' || NEW.id);
  END IF;

  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER handle_notifications_updates_deletes_trigger
AFTER UPDATE OR DELETE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE handle_notifications_updates_deletes();

-- Handle before updates for notifications
CREATE OR REPLACE FUNCTION reset_notification_error_before_update()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.filter != NEW.filter OR OLD.custom_services != NEW.custom_services OR OLD.team_id != NEW.team_id THEN
    NEW.error = NULL;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER reset_notification_error_before_update_trigger
BEFORE UPDATE ON notifications
FOR EACH ROW
EXECUTE PROCEDURE reset_notification_error_before_update();

--- A function to insert only those notifications that were unsent. 
--- It deals with the deduplication of inserting the same notification again if it was silenced or blocked by repeatInterval.
CREATE OR REPLACE FUNCTION insert_unsent_notification_to_history(
  p_notification_id UUID,
  p_source_event TEXT,
  p_resource_id UUID,
  p_status TEXT,
  p_window INTERVAL
) RETURNS VOID AS $$
DECLARE
  v_existing_id UUID;
BEGIN
  IF p_status NOT IN ('silenced', 'repeat-interval') THEN
    RAISE EXCEPTION 'Status must be silenced or repeat-interval';
  END IF;

  SELECT id INTO v_existing_id FROM notification_send_history
  WHERE notification_id = p_notification_id
    AND source_event = p_source_event
    AND resource_id = p_resource_id
    AND status = p_status
    AND created_at > NOW() - p_window
  ORDER BY created_at DESC
  LIMIT 1;
  
  IF v_existing_id IS NOT NULL THEN
    UPDATE notification_send_history SET count = count + 1, created_at = CURRENT_TIMESTAMP
    WHERE id = v_existing_id;
  ELSE
    INSERT INTO notification_send_history (notification_id, status, source_event, resource_id)
    VALUES (p_notification_id, p_status, p_source_event, p_resource_id);
  END IF;
END;
$$ LANGUAGE plpgsql;

--
DROP VIEW IF EXISTS notification_silences_list;

CREATE OR REPLACE VIEW notification_silences_list AS
SELECT 
  notification_silences.id,
  notification_silences.namespace,
  notification_silences.recursive,
  notification_silences.description,
  notification_silences."from",
  notification_silences."until",
  notification_silences.source,
  notification_silences.created_by,
  notification_silences.created_at, 
  notification_silences.updated_at,
  notification_silences.deleted_at,
  COALESCE(config_items.id, components.id, checks.id, canaries.id) as resource_id,
  COALESCE(config_items.type, components.icon, checks.icon) as resource_icon,
  COALESCE(config_items.type, components.type, checks.type) as resource_type,
  COALESCE(config_items.health, components.health, CASE WHEN checks.status = 'passed' THEN 'healthy' ELSE 'unhealthy' END) as resource_health,
  COALESCE(config_items.status, components.status) AS resource_status,
  COALESCE(config_items.name, components.name, checks.name, canaries.name) AS resource_name,
  CASE
    WHEN config_items.id IS NOT NULL THEN 'config_item'
    WHEN components.id IS NOT NULL THEN 'component'
    WHEN checks.id IS NOT NULL THEN 'check'
    WHEN canaries.id IS NOT NULL THEN 'canary'
  END as resource_kind
FROM notification_silences
  LEFT JOIN config_items ON config_items.id = notification_silences.config_id
  LEFT JOIN components ON components.id = notification_silences.component_id
  LEFT JOIN checks ON checks.id = notification_silences.check_id
  LEFT JOIN canaries ON canaries.id = notification_silences.canary_id;
-- 