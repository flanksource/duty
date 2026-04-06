CREATE OR REPLACE VIEW permission_subjects AS
SELECT
  t.id::text AS id,
  t.name,
  'team'::text AS type
FROM teams t
WHERE t.deleted_at IS NULL

UNION ALL

SELECT
  pg.id::text AS id,
  pg.name,
  'permission_subject_group'::text AS type
FROM permission_groups pg
WHERE pg.deleted_at IS NULL
  AND pg.name IS NOT NULL
  AND trim(pg.name) <> ''

UNION ALL

SELECT
  p.id::text AS id,
  p.name,
  'person'::text AS type
FROM people p
WHERE p.deleted_at IS NULL
  AND p.type IS NULL
  AND p.email IS NOT NULL

UNION ALL

SELECT
  r.name AS id,
  r.name,
  'role'::text AS type
FROM (
  VALUES
    ('everyone'),
    ('editor'),
    ('viewer'),
    ('guest')
) AS r(name);