DELETE FROM external_roles
WHERE array_length(aliases, 1) = 1
  AND updated_at IS NULL
  AND role_type IN ('ClusterRole', 'Role')
  AND created_at < COALESCE(
    (
      SELECT MAX(updated_at)
      FROM migration_logs
    ),
    NOW()
  );
