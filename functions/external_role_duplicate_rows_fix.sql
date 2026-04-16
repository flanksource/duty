DO $$
BEGIN
  IF EXISTS (
      SELECT 1
      FROM pg_tables
      WHERE schemaname = 'public'
      AND tablename = 'external_roles'
  ) THEN
      DELETE FROM external_roles WHERE array_length(aliases, 1) = 1 AND updated_at IS NULL AND role_type in ('ClusterRole','Role') AND created_at < '2026-05-01' ;
  END IF;
END $$;
