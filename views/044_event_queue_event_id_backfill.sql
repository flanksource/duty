DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'event_queue'
      AND column_name = 'event_id'
  ) THEN
    WITH raw_candidates AS (
      SELECT
        id,
        name,
        created_at,
        CASE
          WHEN name IN ('config.changed', 'config.updated') THEN properties->>'change_id'
          ELSE properties->>'id'
        END AS event_id_text
      FROM event_queue
      WHERE
        (event_id IS NULL OR event_id = '00000000-0000-0000-0000-000000000000'::uuid)
        AND (
          (name IN ('config.changed', 'config.updated') AND properties ? 'change_id')
          OR
          (name NOT IN ('config.changed', 'config.updated') AND properties ? 'id')
        )
    ),
    -- Only select the first one to satisfy the unique index.
    candidates AS (
      SELECT
        id,
        name,
        event_id_text::uuid AS parsed_event_id,
        ROW_NUMBER() OVER (
          PARTITION BY name, event_id_text::uuid
          ORDER BY created_at DESC, id DESC
        ) AS rn
      FROM raw_candidates
    ),
    to_update AS (
      SELECT c.id, c.parsed_event_id
      FROM candidates c
      WHERE c.rn = 1
        AND NOT EXISTS (
          SELECT 1
          FROM event_queue existing
          WHERE existing.name = c.name
            AND existing.event_id = c.parsed_event_id
        )
    )
    UPDATE event_queue eq
    SET event_id = tu.parsed_event_id
    FROM to_update tu
    WHERE eq.id = tu.id;
  END IF;
END $$;