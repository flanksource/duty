CREATE OR REPLACE VIEW permission_subjects AS
SELECT
  t.id::TEXT AS id,
  t.name,
  'team' AS type,
  NULL AS "email",
  NULL AS "owner"
FROM teams t
WHERE t.deleted_at IS NULL

UNION ALL

SELECT
  pg.id::TEXT AS id,
  pg.name,
  'permission_subject_group' AS type,
  NULL AS "email",
  NULL AS "owner"
FROM permission_groups pg
WHERE pg.deleted_at IS NULL
  AND pg.name IS NOT NULL
  AND trim(pg.name) <> ''

UNION ALL

SELECT
  p.id::TEXT AS id,
  access_tokens.name,
  'access_token_person' AS type,
  NULL AS "email",
  access_tokens.created_by::TEXT AS "owner"
FROM people p
INNER JOIN access_tokens
  ON p.id = access_tokens.person_id
WHERE p.deleted_at IS NULL
  AND p.type = 'access_token'
  AND p.email IS NOT NULL

UNION ALL

SELECT
  p.id::TEXT AS id,
  p.name,
  'person' AS type,
  p.email,
  NULL AS "owner"
FROM people p
WHERE p.deleted_at IS NULL
  AND p.type IS NULL
  AND p.email IS NOT NULL

UNION ALL

SELECT
  r.name::TEXT AS id,
  r.name,
  'role' AS type,
  NULL AS "email",
  NULL AS "owner"
FROM (
  VALUES
    ('everyone'),
    ('editor'),
    ('viewer'),
    ('guest')
) AS r(name);