DO $$
BEGIN
    IF NOT EXISTS (select pg_column_compression(details) as pg_column_compression from check_statuses where pg_column_compression(details) = 'pglz' group by 1) THEN
        ALTER TABLE check_statuses ALTER COLUMN  details SET COMPRESSION lz4
    END IF;
END $$;
