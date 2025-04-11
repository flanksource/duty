-- runs: always
-- Always run this to prevent queries from failing after an upgrade.
-- It's idempotent, so it's safe to run multiple times.

DO $$
DECLARE
  v_major_version integer;
BEGIN
  SELECT current_setting('server_version_num')::integer INTO v_major_version;

  IF v_major_version >= 150000 THEN
    DROP INDEX IF EXISTS unique_notification_group_resources_unresolved_config;
    DROP INDEX IF EXISTS unique_notification_group_resources_unresolved_check;
    DROP INDEX IF EXISTS unique_notification_group_resources_unresolved_component;
    
    -- use EXECUTE to avoid error during parsing in PostgreSQL versions < 15,
    -- since NULLS NOT DISTINCT is only supported starting in Postgres 15.
    EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS unique_notification_group_resources_unresolved
    ON public.notification_group_resources (group_id, config_id, check_id, component_id)
    NULLS NOT DISTINCT
    WHERE resolved_at IS NULL';
  ELSE
    DROP INDEX IF EXISTS unique_notification_group_resources_unresolved;
    
    CREATE UNIQUE INDEX IF NOT EXISTS unique_notification_group_resources_unresolved_config
    ON public.notification_group_resources (group_id, config_id)
    WHERE resolved_at IS NULL AND config_id IS NOT NULL;

    CREATE UNIQUE INDEX IF NOT EXISTS unique_notification_group_resources_unresolved_check
    ON public.notification_group_resources (group_id, check_id)
    WHERE resolved_at IS NULL AND check_id IS NOT NULL;

    CREATE UNIQUE INDEX IF NOT EXISTS unique_notification_group_resources_unresolved_component
    ON public.notification_group_resources (group_id, component_id)
    WHERE resolved_at IS NULL AND component_id IS NOT NULL;
  END IF;
END;
$$ LANGUAGE plpgsql;