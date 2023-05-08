DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'postgrest_api') THEN
        CREATE ROLE postgrest_api;
        GRANT SELECT, UPDATE, DELETE, INSERT ON ALL TABLES IN SCHEMA public TO postgrest_api;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, UPDATE, DELETE, INSERT ON TABLES TO postgrest_api;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO postgrest_api;
    END IF;

    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'postgrest_anon') THEN
        CREATE ROLE postgrest_anon;
    END IF;
END $$;


-- We need to reload postgrest schema after DDL Updates
-- Docs: https://postgrest.org/en/stable/schema_cache.html

-- Create an event trigger function
CREATE OR REPLACE FUNCTION public.pgrst_watch()
RETURNS event_trigger AS $$
BEGIN
    NOTIFY pgrst, 'reload schema';
END
$$ LANGUAGE plpgsql;

-- This event trigger will fire after every ddl_command_end event
DROP EVENT TRIGGER IF EXISTS pgrst_watch;
CREATE EVENT TRIGGER pgrst_watch
    ON ddl_command_end
    EXECUTE PROCEDURE public.pgrst_watch();
