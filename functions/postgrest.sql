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
END $$;


DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'postgrest_api') THEN
      -- CREATE a ROLE that will own all views where we need to enforce RLS.
      CREATE ROLE api_views_owner NOSUPERUSER NOBYPASSRLS;

      GRANT SELECT ON ALL TABLES IN SCHEMA public TO api_views_owner;
    END IF ;
END
$$;

