DO $$
BEGIN
    IF to_regclass('public.config_relationships') IS NOT NULL THEN
        WITH ranked AS (
            SELECT
                ctid,
                ROW_NUMBER() OVER (
                    PARTITION BY related_id, config_id, relation, scraper_id
                    ORDER BY (deleted_at IS NULL) DESC, updated_at DESC, created_at DESC
                ) AS row_number
            FROM config_relationships
        )
        DELETE FROM config_relationships
        USING ranked
        WHERE config_relationships.ctid = ranked.ctid
          AND ranked.row_number > 1;
    END IF;
END $$;
