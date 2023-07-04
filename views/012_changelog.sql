-- Push table changes to event queue
CREATE
OR REPLACE FUNCTION push_changes_to_event_queue () RETURNS trigger AS $$
DECLARE
    rec RECORD;
    payload JSONB;
    priority integer := 0;
BEGIN
  rec = NEW;
  IF TG_OP = 'DELETE' THEN
    -- Do not push deletions in event queue
    return NULL;
  END IF;

  CASE TG_TABLE_NAME
    WHEN 'component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'relationship_id', rec.relationship_id, 'selector_id', rec.selector_id);
    WHEN 'config_component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'config_id', rec.config_id);
    WHEN 'config_relationships' THEN
      payload = jsonb_build_object('related_id', rec.related_id, 'config_id', rec.config_id, 'selector_id', rec.selector_id);
    WHEN 'check_statuses' THEN
      payload = jsonb_build_object('check_id', rec.check_id, 'time', rec.time);
    WHEN 'checks' THEN
      -- Set these fields to null for checks to prevent excessive pushes
      rec.last_runtime = NULL;
      rec.last_transition_time = NULL;
      rec.updated_at = NULL;
      OLD.last_runtime = NULL;
      OLD.last_transition_time = NULL;
      OLD.updated_at = NULL;

      -- If it is same as the old record, then no action required
      IF rec IS NOT DISTINCT FROM OLD THEN
        RETURN NEW;
      END IF;
      priority = 10;
      payload = jsonb_build_object('id', rec.id);
    ELSE
      priority = 10;
      payload = jsonb_build_object('id', rec.id);
  END CASE;

  INSERT INTO
    event_queue (name, properties, priority)
  VALUES
    ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME) || payload, priority)
  ON CONFLICT
    (name, properties)
  DO UPDATE SET
    attempts = 0;
  NOTIFY event_queue_updates, 'update';
  RETURN NULL;
END;
$$ LANGUAGE 'plpgsql' SECURITY DEFINER;

-- Apply trigger
DO $$ 
DECLARE 
  table_name TEXT;
BEGIN 
  FOR table_name IN 
    SELECT t.table_name 
    FROM information_schema.tables  t
    WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
      AND t.table_name IN (
        'canaries',
        'checks',
        'components',
        'config_scrapers',
        'config_analysis',
        'config_changes',
        'config_items',
        'check_statuses',
        'config_component_relationships',
        'component_relationships',
        'config_relationships'
      )
  LOOP 
    EXECUTE format('
      CREATE OR REPLACE TRIGGER %1$I_change_to_event_queue
      AFTER INSERT OR UPDATE ON %1$I
      FOR EACH ROW
      EXECUTE PROCEDURE push_changes_to_event_queue()',
      table_name
    );
  END LOOP; 
END $$;
