CREATE OR REPLACE VIEW permission_subjects AS
SELECT
  t.id::TEXT AS id,
  t.name,
  'team' AS type,
  NULL AS "email",
  NULL AS "owner",
  NULL AS "icon"
FROM teams t
WHERE t.deleted_at IS NULL

UNION ALL

SELECT
  p.id::TEXT AS id,
  p.name,
  'person' AS type,
  p.email,
  NULL AS "owner",
  NULL AS "icon"
FROM people p
WHERE p.deleted_at IS NULL
  AND p.type IS NULL
  AND p.email IS NOT NULL

UNION ALL

SELECT
  pb.id::TEXT AS id,
  COALESCE(NULLIF(pb.title, ''), pb.name) AS name,
  'playbook' AS type,
  NULL AS "email",
  pb.created_by::TEXT AS "owner",
  pb.icon AS "icon"
FROM playbooks pb
WHERE pb.deleted_at IS NULL;
