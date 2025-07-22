CREATE OR REPLACE VIEW views_summary AS
SELECT 
    id,
    namespace,
    name,
    spec->>'title' AS title,
    spec->>'icon' AS icon,
    last_ran
FROM views
WHERE deleted_at IS NULL;