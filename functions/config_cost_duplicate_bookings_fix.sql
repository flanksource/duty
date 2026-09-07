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
-- than adding to it — so the duplicate is dropped, never summed.
--
-- Which copy survives is the whole point. Both are written by the same upsert, so they
-- share updated_at and created_at; ordering by those alone leaves the id to decide, and a
-- ULID orders by insertion, which says nothing about where the money belongs. The copy
-- booked against the config item that actually carries the charge's resource id is
-- therefore preferred, and the rest of the order only settles copies that are alike in
-- that respect. Keeping the root copy over the resource copy would delete the attribution
-- this table exists to record, and no later scrape restores it: a charge is re-resolved
-- only while its billing period is still being restated.
DO $$
DECLARE
    target text;
BEGIN
    FOREACH target IN ARRAY ARRAY['config_costs', 'config_cost_compact'] LOOP
        CONTINUE WHEN to_regclass('public.' || target) IS NULL;

        -- Only duplicated charges are ranked. The catalog lookup is a probe per row, so
        -- confining it to the groups that have a duplicate keeps this off the whole
        -- retained history.
        EXECUTE format($fmt$
            WITH duplicated AS (
                SELECT source_key, fingerprint, period_start, period_end
                FROM %1$I
                GROUP BY 1, 2, 3, 4
                HAVING count(*) > 1
            ), candidates AS (
                SELECT c.ctid, c.source_key, c.fingerprint, c.period_start, c.period_end,
                       c.updated_at, c.created_at, c.id,
                       (c.external_id IS NOT NULL AND EXISTS (
                           SELECT 1 FROM config_items ci
                           WHERE ci.id = c.config_id
                             AND ci.external_id @> ARRAY[c.external_id])) AS names_resource
                FROM %1$I c
                JOIN duplicated d
                  ON d.source_key = c.source_key
                 AND d.fingerprint = c.fingerprint
                 AND d.period_start = c.period_start
                 AND d.period_end = c.period_end
            ), ranked AS (
                SELECT ctid,
                       ROW_NUMBER() OVER (
                           PARTITION BY source_key, fingerprint, period_start, period_end
                           ORDER BY names_resource DESC, updated_at DESC, created_at DESC, id DESC
                       ) AS row_number
                FROM candidates
            )
            DELETE FROM %1$I
            USING ranked
            WHERE %1$I.ctid = ranked.ctid
              AND ranked.row_number > 1
        $fmt$, target);
    END LOOP;
END $$;
