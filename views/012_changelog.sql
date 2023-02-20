-- Trigger def
CREATE OR REPLACE FUNCTION change_trigger() 
RETURNS trigger AS $$
BEGIN
  IF TG_TABLE_NAME = 'component_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(NEW.component_id, ':', NEW.relationship_id, ':', NEW.selector_id));
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(OLD.component_id, ':', OLD.relationship_id, ':', OLD.selector_id));
      RETURN OLD;
    END IF;

  ELSIF TG_TABLE_NAME = 'config_component_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(NEW.component_id, ':', NEW.config_id));
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(OLD.component_id, ':', OLD.config_id));
      RETURN OLD;
    END IF;
  
  ELSIF TG_TABLE_NAME = 'config_relationships' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(NEW.related_id, ':', NEW.config_id, ':', NEW.selector_id));
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(OLD.related_id, ':', OLD.config_id, ':', OLD.selector_id));
      RETURN OLD;
    END IF;

  ELSIF TG_TABLE_NAME = 'check_statuses' THEN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(NEW.check_id, ':', NEW.time));
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, CONCAT(OLD.check_id, ':', NEW.time));
      RETURN OLD;
    END IF;
  
  ELSE
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, NEW.id);
      RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
      INSERT INTO push_queue (table_name, operation, item_id) VALUES (TG_RELNAME, TG_OP, OLD.id);
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
