DO $$
BEGIN
  IF to_regclass('public.playbooks') IS NOT NULL THEN
    WITH ranked AS (
      SELECT
        id,
        ROW_NUMBER() OVER (
          PARTITION BY spec->'on'->'webhook'->>'path'
          ORDER BY created_at, id
        ) AS row_number
      FROM playbooks
      WHERE deleted_at IS NULL
        AND COALESCE(spec->'on'->'webhook'->>'path', '') <> ''
    )
    UPDATE playbooks
    SET spec = jsonb_set(spec, '{on,webhook,path}', '""'::jsonb)
    FROM ranked
    WHERE playbooks.id = ranked.id
      AND ranked.row_number > 1;
  END IF;
END $$;
