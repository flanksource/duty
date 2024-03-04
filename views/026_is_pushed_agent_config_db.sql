DO $$ 
DECLARE 
  table_name TEXT;
BEGIN 
  FOR table_name IN 
    SELECT t.table_name 
    FROM information_schema.tables  t
    WHERE t.table_schema = current_schema() AND t.table_type = 'BASE TABLE'
      AND t.table_name IN (
        'config_scrapers',
        'config_items',
        'config_changes',
        'config_analysis'
      )
  LOOP 
    EXECUTE format('
      CREATE OR REPLACE TRIGGER %I_reset_is_pushed_before_update
      BEFORE UPDATE ON %I
      FOR EACH ROW
      EXECUTE PROCEDURE reset_is_pushed_before_update()',
      table_name, table_name
    );
  END LOOP; 
END $$;