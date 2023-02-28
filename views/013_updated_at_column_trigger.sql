CREATE EXTENSION IF NOT EXISTS hstore;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
DECLARE
  changed_fields hstore;
BEGIN
  IF NOT (TG_WHEN = 'BEFORE' AND TG_OP = 'UPDATE') THEN
    RAISE EXCEPTION 'update_updated_at_column() should only run as a BEFORE UPDATE trigger';
  END IF;

  changed_fields = hstore(NEW.*) - hstore(OLD.*);
  IF changed_fields = hstore('') THEN
    RETURN NULL; -- No columns have been updated.
  END IF;

  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Iterate over all tables (excluding views) in the current schema and 
-- create a trigger on each table that has an "updated_at" column
DO $$ 
DECLARE 
  tbl_name TEXT;
BEGIN
  FOR tbl_name IN
    SELECT table_name
    FROM information_schema.columns
    WHERE 
      table_schema = current_schema() 
      AND column_name = 'updated_at'
      AND table_name NOT IN (SELECT table_name FROM information_schema.views WHERE table_schema = current_schema()) 
  LOOP
    EXECUTE format('CREATE OR REPLACE TRIGGER %I_update_updated_at
      BEFORE UPDATE ON %I
      FOR EACH ROW
      EXECUTE FUNCTION update_updated_at_column()', tbl_name, tbl_name);
  END LOOP;
END $$;