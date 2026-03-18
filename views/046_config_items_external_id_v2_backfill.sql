-- STEP-1 migration backfill for config_items external IDs.
--
-- Purpose:
--   - Keep legacy config_items.external_id (text[]) unchanged.
--   - Populate canonical config_items.external_id_v2 from external_id[1].
--   - Populate config_items.aliases from external_id[2:] as distinct lowercase aliases.
--
-- Safety / idempotency:
--   - Runs only when required columns exist.
--   - Fails fast if active canonical values would violate uniqueness.
--   - Updates only rows whose target values differ (IS DISTINCT FROM), so reruns are safe.
--
-- PostgreSQL array note:
--   - Arrays are 1-based; canonical value is external_id[1], aliases come from external_id[2:].
DO $$
DECLARE
  duplicate_values TEXT;
BEGIN
  -- Guard: skip gracefully if schema has not yet added the new columns.
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config_items'
      AND column_name = 'external_id'
  )
  AND EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config_items'
      AND column_name = 'external_id_v2'
  )
  AND EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'config_items'
      AND column_name = 'aliases'
  ) THEN
    -- Precheck: active-row duplicate canonical IDs are not allowed for v2 unique index.
    SELECT string_agg(
      format('%s (%s rows)', canonical_external_id, dup_count),
      ', '
      ORDER BY dup_count DESC, canonical_external_id
    )
    INTO duplicate_values
    FROM (
      SELECT lower(external_id[1]) AS canonical_external_id, COUNT(*) AS dup_count
      FROM config_items
      WHERE deleted_at IS NULL
        AND array_length(external_id, 1) >= 1
        AND external_id[1] IS NOT NULL
      GROUP BY lower(external_id[1])
      HAVING COUNT(*) > 1
      ORDER BY COUNT(*) DESC, lower(external_id[1])
      LIMIT 20
    ) dupes;

    IF duplicate_values IS NOT NULL THEN
      RAISE EXCEPTION 'config_items.external_id_v2 backfill aborted: duplicate canonical external IDs found among active rows (%). Resolve duplicates in config_items.external_id[1] before rerunning.', duplicate_values;
    END IF;

    -- Backfill canonical ID and aliases from the legacy array column.
    WITH computed AS (
      SELECT
        id,
        lower(external_id[1]) AS new_external_id_v2,
        NULLIF(
          ARRAY(
            SELECT DISTINCT lower(alias_value)
            FROM unnest(COALESCE(external_id[2:], '{}'::text[])) AS alias_value
            WHERE alias_value IS NOT NULL
              AND btrim(alias_value) <> ''
            ORDER BY lower(alias_value)
          ),
          '{}'::text[]
        ) AS new_aliases
      FROM config_items
    )
    UPDATE config_items ci
    SET external_id_v2 = computed.new_external_id_v2,
        aliases = computed.new_aliases
    FROM computed
    WHERE ci.id = computed.id
      AND (
        ci.external_id_v2 IS DISTINCT FROM computed.new_external_id_v2
        OR ci.aliases IS DISTINCT FROM computed.new_aliases
      );
  END IF;
END $$;
