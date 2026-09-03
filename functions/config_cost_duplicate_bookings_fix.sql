-- Retires cost bookings left behind when a charge's attribution moved.
--
-- config_costs and config_cost_compact used to merge on a key that included config_id.
-- A charge is re-resolved on every scrape, and moves off the account root onto its own
-- resource as soon as the catalog discovers that resource — so the move inserted a second
-- row rather than updating the first, and both were then summed. Every resource
-- discovered after its costs first landed carries one of these.
--
-- The merge key no longer includes config_id, so this clears the rows that accumulated
-- while it did. It must run before that unique index is built, which is why it lives here
-- rather than in views/: functions run ahead of the schema apply.
--
-- Both copies of a charge carry the same amount — the upsert always SET the total rather
-- than adding to it — so the duplicate is dropped, never summed. The surviving row is the
-- most recently resolved one, matching the rule the ingest path now follows.
DO $$
BEGIN
    IF to_regclass('public.config_costs') IS NOT NULL THEN
        WITH ranked AS (
            SELECT
                ctid,
                ROW_NUMBER() OVER (
                    PARTITION BY source_key, fingerprint, period_start, period_end
                    ORDER BY updated_at DESC, created_at DESC, id DESC
                ) AS row_number
            FROM config_costs
        )
        DELETE FROM config_costs
        USING ranked
        WHERE config_costs.ctid = ranked.ctid
          AND ranked.row_number > 1;
    END IF;

    IF to_regclass('public.config_cost_compact') IS NOT NULL THEN
        WITH ranked AS (
            SELECT
                ctid,
                ROW_NUMBER() OVER (
                    PARTITION BY source_key, fingerprint, period_start, period_end
                    ORDER BY updated_at DESC, created_at DESC, id DESC
                ) AS row_number
            FROM config_cost_compact
        )
        DELETE FROM config_cost_compact
        USING ranked
        WHERE config_cost_compact.ctid = ranked.ctid
          AND ranked.row_number > 1;
    END IF;
END $$;
