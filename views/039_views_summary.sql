DROP VIEW IF EXISTS views_summary;

CREATE OR REPLACE VIEW views_summary AS
SELECT 
    id,
    namespace,
    name,
    spec->'display'->>'title' AS title,
    spec->'display'->>'icon' AS icon,
    (spec->'display'->>'ordinal')::int AS ordinal,
    (spec->'display'->>'sidebar')::boolean AS sidebar
FROM views
WHERE deleted_at IS NULL;