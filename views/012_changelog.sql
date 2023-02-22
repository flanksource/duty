-- Trigger def
CREATE OR REPLACE FUNCTION change_trigger() 
RETURNS trigger AS $$
BEGIN
  IF TG_TABLE_NAME = 'component_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', NEW.component_id, 'relationship_id', NEW.relationship_id, 'selector_id', NEW.selector_id));
      NOTIFY event_queue_updates, 'update';
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', OLD.component_id, 'relationship_id', OLD.relationship_id, 'selector_id', OLD.selector_id));
      NOTIFY event_queue_updates, 'update';
      RETURN OLD;
    END IF;

  ELSIF TG_TABLE_NAME = 'config_component_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', NEW.component_id, 'config_id', NEW.config_id));
      NOTIFY event_queue_updates, 'update';
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'component_id', OLD.component_id, 'config_id', OLD.config_id));
      NOTIFY event_queue_updates, 'update';
      RETURN OLD;
    END IF;
  
  ELSIF TG_TABLE_NAME = 'config_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'related_id', NEW.related_id, 'config_id', NEW.config_id, 'selector_id', NEW.selector_id));
      NOTIFY event_queue_updates, 'update';
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'related_id', OLD.related_id, 'config_id', OLD.config_id, 'selector_id', OLD.selector_id));
      NOTIFY event_queue_updates, 'update';
      RETURN OLD;
    END IF;

  ELSIF TG_TABLE_NAME = 'check_statuses' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'check_id', NEW.check_id, 'time', NEW.time));
      NOTIFY event_queue_updates, 'update';
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'check_id', OLD.check_id, 'time', OLD.time));
      NOTIFY event_queue_updates, 'update';
      RETURN OLD;
    END IF;
  
  ELSE
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'id', NEW.id));
      NOTIFY event_queue_updates, 'update';
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO event_queue (name, properties) VALUES ('push_queue.create', jsonb_build_object('table', TG_TABLE_NAME, 'id', OLD.id));
      NOTIFY event_queue_updates, 'update';
      RETURN OLD;
    END IF;
  END IF;
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
    EXECUTE format(
      'DROP TRIGGER IF EXISTS %1$I_change_trigger ON %1$I;
      CREATE TRIGGER %1$I_change_trigger
      BEFORE INSERT OR UPDATE OR DELETE ON %1$I
      FOR EACH ROW
      EXECUTE PROCEDURE change_trigger()',
      table_name
    );
  END LOOP; 
END $$;
