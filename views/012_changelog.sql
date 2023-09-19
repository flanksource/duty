-- Push table changes to event queue
CREATE
OR REPLACE FUNCTION push_changes_to_event_queue () RETURNS trigger AS $$
DECLARE
    rec RECORD;
    payload JSONB;
    priority integer := 0;
    priority_table JSONB := '{
        "topologies": 20,
        "canaries": 20,
        "config_scrapers": 20,
        "checks": 10,
        "components": 10,
        "config_items": 10,
        "config_analysis": 5,
        "config_changes": 5,
        "config_component_relationships": 5,
        "component_relationships": 5,
        "config_relationships": 5,
        "check_statuses": 0
    }';
BEGIN
  rec = NEW;
  IF TG_OP = 'DELETE' THEN
    -- Do not push deletions in event queue
    return NULL;
  END IF;

  CASE TG_TABLE_NAME
    WHEN 'component_relationships' THEN
      -- Set these fields to null for component_relationships to prevent excessive pushes
      rec.updated_at = NULL;
      OLD.updated_at = NULL;

      -- If it is same as the old record, then no action required
      IF rec IS NOT DISTINCT FROM OLD THEN
        RETURN NULL;
      END IF;

      payload = jsonb_build_object('component_id', rec.component_id, 'relationship_id', rec.relationship_id, 'selector_id', rec.selector_id);
    WHEN 'config_component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'config_id', rec.config_id);
    WHEN 'config_relationships' THEN
      payload = jsonb_build_object('related_id', rec.related_id, 'config_id', rec.config_id, 'selector_id', rec.selector_id);
    WHEN 'check_statuses' THEN
      payload = jsonb_build_object('check_id', rec.check_id, 'time', rec.time);
    WHEN 'checks' THEN
      -- Set these fields to null for checks to prevent excessive pushes
      rec.updated_at = NULL;
      OLD.updated_at = NULL;

      -- If it is same as the old record, then no action required
      IF rec IS NOT DISTINCT FROM OLD THEN
        RETURN NULL;
      END IF;
      payload = jsonb_build_object('id', rec.id);
    WHEN 'canaries' THEN
      -- Set these fields to null for canaries to prevent excessive pushes
      rec.updated_at = NULL;
      OLD.updated_at = NULL;

      -- If it is same as the old record, then no action required
      IF rec IS NOT DISTINCT FROM OLD THEN
        RETURN NULL;
      END IF;
      payload = jsonb_build_object('id', rec.id);
    ELSE
      payload = jsonb_build_object('id', rec.id);
  END CASE;

  -- Log changes to event queue
  priority = (priority_table->>TG_TABLE_NAME)::integer;
  INSERT INTO event_queue (name, properties, priority) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME) || payload, priority)
  ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;

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
        'config_relationships',
        'topologies'
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
