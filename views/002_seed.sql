DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM people WHERE name = 'System') THEN
        INSERT INTO people (name) VALUES ('System');
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM agents WHERE name = 'local') THEN
        INSERT INTO agents (id, name) VALUES ('00000000-0000-0000-0000-000000000000', 'local');
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM config_scrapers WHERE name = 'System') THEN
        INSERT INTO config_scrapers (id, name, source, spec)
        VALUES ('00000000-0000-0000-0000-000000000000', 'System', 'System', '{"schedule": "@every 5m", "system": true}');
    END IF;
END $$;

DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM severities ) THEN
        INSERT INTO severities (id, name, icon, aliases)
        VALUES
            (1, 'Critical', 'error',ARRAY ['P1']),
            (2, 'Blocker', 'error', ARRAY['P2']),
            (3, 'High', 'warning',ARRAY ['P3']),
            (4, 'Medium', 'info',ARRAY ['P4']),
            (5, 'Low', 'info', ARRAY['P4']);
   END IF;
END $$;

-- These columns are generated via a script instead of being defined through atlas schemas,
-- because atlas fails to generate an accurate diff for them.
-- it always detects a change in the expression.
DO $$
DECLARE
  tbl TEXT;
  tables TEXT[] := ARRAY['components', 'config_items'];
BEGIN
  FOREACH tbl IN ARRAY tables LOOP
    IF NOT EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_name = tbl AND column_name = 'properties_values'
    ) THEN
      EXECUTE format('
          ALTER TABLE %I
          ADD COLUMN properties_values JSONB
          GENERATED ALWAYS AS (
              CASE
                  WHEN properties IS NULL THEN NULL
                  ELSE jsonb_path_query_array(properties, ''$[*].text''::jsonpath) ||
                        jsonb_path_query_array(properties, ''$[*].value''::jsonpath)
              END
          ) STORED;
      ', tbl);
    END IF;

    IF NOT EXISTS (
      SELECT 1
      FROM pg_indexes
      WHERE tablename = tbl AND indexname = tbl || '_properties_values_gin_idx'
    ) THEN
        EXECUTE format('CREATE INDEX %I ON %I USING GIN (properties_values)', tbl || '_properties_values_gin_idx', tbl);
    END IF;
  END LOOP;
END $$;
