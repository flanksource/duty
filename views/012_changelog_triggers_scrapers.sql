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
        'config_items'
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
