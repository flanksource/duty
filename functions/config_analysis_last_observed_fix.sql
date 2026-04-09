DO $$
BEGIN
  IF EXISTS (
      SELECT 1
      FROM pg_tables
      WHERE schemaname = 'public'
      AND tablename = 'config_analysis'
  ) THEN
      UPDATE config_analysis SET last_observed = NOW() WHERE last_observed IS NULL;
  END IF;
END $$;
