-- Push table changes to event queue
CREATE
OR REPLACE FUNCTION push_changes_to_event_queue () RETURNS trigger AS $$
DECLARE
    rec RECORD;
BEGIN
  rec = NEW;
  IF TG_OP = 'DELETE' THEN
    rec = OLD;
  END IF;

  IF TG_TABLE_NAME = 'component_relationships' THEN
    INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', rec.component_id, 'relationship_id', rec.relationship_id, 'selector_id', rec.selector_id)) ON CONFLICT (name, properties) DO UPDATE SET attempts = 0;
    RETURN rec;

  ELSIF TG_TABLE_NAME = 'config_component_relationships' THEN
    INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', rec.component_id, 'config_id', rec.config_id)) ON CONFLICT (name, properties) DO UPDATE SET attempts = 0;
    RETURN rec;

  ELSIF TG_TABLE_NAME = 'config_relationships' THEN
    INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'related_id', rec.related_id, 'config_id', rec.config_id, 'selector_id', rec.selector_id)) ON CONFLICT (name, properties) DO UPDATE SET attempts = 0;
    RETURN rec;

  ELSIF TG_TABLE_NAME = 'check_statuses' THEN
    INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'check_id', rec.check_id, 'time', rec.time)) ON CONFLICT (name, properties) DO UPDATE SET attempts = 0;
    RETURN rec;
  
  ELSE
    INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'id', rec.id)) ON CONFLICT (name, properties) DO UPDATE SET attempts = 0;
    RETURN rec;
  END IF;

  NOTIFY event_queue_updates, 'update';
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
      BEFORE INSERT OR UPDATE OR DELETE ON %1$I
      FOR EACH ROW
      EXECUTE PROCEDURE push_changes_to_event_queue()',
      table_name
    );
  END LOOP; 
END $$;
