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
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO postgrest_anon;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO postgrest_anon;
    END IF;

    IF current_setting('server_version_num')::int >= 160000
        AND NOT EXISTS (
            SELECT FROM pg_catalog.pg_roles
            WHERE rolname = current_user AND rolsuper
        )
    THEN
        EXECUTE format('GRANT %I TO %I WITH SET TRUE, INHERIT FALSE', 'postgrest_api', current_user);
        EXECUTE format('GRANT %I TO %I WITH SET TRUE, INHERIT FALSE', 'postgrest_anon', current_user);
    END IF;
END $$;
