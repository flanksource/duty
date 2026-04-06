CREATE OR REPLACE VIEW artifact_summary AS
SELECT
  content_type,
  CASE WHEN a.content IS NOT NULL THEN 'inline' ELSE 'external' END AS storage,
  a.connection_id,
  c.name AS connection_name,
  c.type AS connection_type,
  COUNT(*) AS total_count,
  COALESCE(SUM(a.size), 0) AS total_size
FROM artifacts a
LEFT JOIN connections c ON a.connection_id = c.id
WHERE a.deleted_at IS NULL
GROUP BY
  content_type,
  (CASE WHEN a.content IS NOT NULL THEN 'inline' ELSE 'external' END),
  a.connection_id,
  c.name,
  c.type;
