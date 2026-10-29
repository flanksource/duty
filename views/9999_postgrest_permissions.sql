-- runs: always
-- Run last on every migration: granting access again must not undo these restrictions.
-- The migration runner commits the grants and revokes together.
DO $$
DECLARE
    api_roles text[] := ARRAY['postgrest_api', current_setting('duty.postgrest_api_role', true)];
    anon_roles text[] := ARRAY['postgrest_anon', current_setting('duty.postgrest_anon_role', true)];
    -- Only the backend may use these tables, never PostgREST.
    internal_tables text[] := ARRAY[
        'notification_health_states',
        'notification_health_episodes',
        'notification_deliveries',
        'connections',
        'properties',
        -- Credentials and authorization state. PostgREST can still reach a table it doesn't serve by embedding it.
        'access_tokens',
        'casbin_rule',
        'invites',
        'oidc_auth_requests',
        'oidc_public_keys',
        'oidc_refresh_tokens'
    ];
    internal_functions text[] := ARRAY[
        'record_notification_health(text,uuid,text)',
        'refresh_notification_health(text,uuid)',
        'notification_health_source_trigger()',
        'insert_check_updates_in_event_queue()',
        'insert_config_health_updates_in_event_queue()',
        'insert_component_health_updates_in_event_queue()',
        'scope_rebuild_members(uuid)'
    ];
    role_name text;
    grantee text;
    table_name text;
    function_name text;
    owned_tables text;
    owned_sequences text;
    owned_functions text;
BEGIN
    -- Grant only on objects the migrating user owns, not on ones owned by other roles or by extensions,
    -- e.g. pg_stat_statements installed in public by a superuser. GRANT ... ON ALL ... fails on those.
    SELECT string_agg(c.oid::regclass::text, ', ') FILTER (WHERE c.relkind IN ('r', 'v', 'm', 'f', 'p')),
           string_agg(c.oid::regclass::text, ', ') FILTER (WHERE c.relkind = 'S')
    INTO owned_tables, owned_sequences
    FROM pg_class c
    WHERE c.relnamespace = 'public'::regnamespace
        AND pg_has_role(c.relowner, 'USAGE')
        AND NOT EXISTS (
            SELECT 1 FROM pg_depend d
            WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
        );

    SELECT string_agg(p.oid::regprocedure::text, ', ')
    INTO owned_functions
    FROM pg_proc p
    WHERE p.pronamespace = 'public'::regnamespace
        AND p.prokind <> 'p'
        AND pg_has_role(p.proowner, 'USAGE')
        AND NOT EXISTS (
            SELECT 1 FROM pg_depend d
            WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e'
        );

    -- Grant normal access first, then remove access to the sensitive objects below.
    FOREACH role_name IN ARRAY api_roles LOOP
        IF role_name IS NULL OR role_name = '' THEN CONTINUE; END IF;
        IF owned_tables IS NOT NULL THEN
            EXECUTE format('GRANT ALL PRIVILEGES ON TABLE %s TO %I', owned_tables, role_name);
        END IF;
        IF owned_sequences IS NOT NULL THEN
            EXECUTE format('GRANT USAGE, SELECT ON SEQUENCE %s TO %I', owned_sequences, role_name);
        END IF;
        IF owned_functions IS NOT NULL THEN
            EXECUTE format('GRANT EXECUTE ON ROUTINE %s TO %I', owned_functions, role_name);
        END IF;
        -- Tables created later, such as generated view tables, need access too.
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %I', role_name);
    END LOOP;

    FOREACH role_name IN ARRAY anon_roles LOOP
        IF role_name IS NULL OR role_name = '' THEN CONTINUE; END IF;
        IF owned_tables IS NOT NULL THEN
            EXECUTE format('GRANT SELECT ON TABLE %s TO %I', owned_tables, role_name);
        END IF;
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO %I', role_name);
    END LOOP;

    -- PUBLIC means every database role, so leaving its access would bypass these restrictions.
    FOREACH role_name IN ARRAY ARRAY['PUBLIC'] || api_roles || anon_roles LOOP
        IF role_name IS NULL OR role_name = '' THEN CONTINUE; END IF;
        grantee := CASE WHEN role_name = 'PUBLIC' THEN 'PUBLIC' ELSE format('%I', role_name) END;

        FOREACH table_name IN ARRAY internal_tables LOOP
            EXECUTE format('REVOKE ALL ON TABLE public.%I FROM %s', table_name, grantee);
        END LOOP;

        FOREACH function_name IN ARRAY internal_functions LOOP
            IF to_regprocedure('public.' || function_name) IS NOT NULL THEN
                EXECUTE format('REVOKE ALL ON FUNCTION public.%s FROM %s', function_name, grantee);
            END IF;
        END LOOP;

        -- A simple view can write to its source table. Allow reads here, but no writes.
        EXECUTE format('REVOKE ALL ON public.feature_flags FROM %s', grantee);
        IF role_name <> 'PUBLIC' THEN
            EXECUTE format('GRANT SELECT ON public.feature_flags TO %I', role_name);
        END IF;
    END LOOP;
END $$;
