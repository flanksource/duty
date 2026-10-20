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
BEGIN
    -- Grant normal access first, then remove access to the sensitive objects below.
    FOREACH role_name IN ARRAY api_roles LOOP
        IF role_name IS NULL OR role_name = '' THEN CONTINUE; END IF;
        EXECUTE format('GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %I', role_name);
        EXECUTE format('GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %I', role_name);
        EXECUTE format('GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO %I', role_name);
        -- Tables created later, such as generated view tables, need access too.
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO %I', role_name);
    END LOOP;

    FOREACH role_name IN ARRAY anon_roles LOOP
        IF role_name IS NULL OR role_name = '' THEN CONTINUE; END IF;
        EXECUTE format('GRANT SELECT ON ALL TABLES IN SCHEMA public TO %I', role_name);
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
