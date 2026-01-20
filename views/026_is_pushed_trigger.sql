CREATE
OR REPLACE FUNCTION reset_is_pushed_before_update() RETURNS TRIGGER AS $$
BEGIN
  -- If any column other than is_pushed is changed, reset is_pushed to false.
  IF NEW IS DISTINCT FROM OLD AND NEW.is_pushed IS NOT DISTINCT FROM OLD.is_pushed THEN
    NEW.is_pushed = false;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
DECLARE
  table_name TEXT;
BEGIN
  FOR table_name IN
    SELECT t.table_name
    FROM information_schema.tables  t
    WHERE t.table_schema = current_schema() AND t.table_type = 'BASE TABLE'
      AND t.table_name IN (
        'artifacts',
        'topologies',
        'config_scrapers',
        'canaries',
        'components',
        'checks',
        'checks_unlogged',
        'config_items',
        'config_items_last_scraped_time',
        'config_analysis',
        'config_changes',
        'check_statuses',
        'check_component_relationships',
        'check_config_relationships',
        'component_relationships',
        'config_component_relationships',
        'config_relationships'
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
