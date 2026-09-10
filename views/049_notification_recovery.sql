-- Health markers are committed with source health changes, not with coalesced events.
CREATE OR REPLACE FUNCTION public.record_notification_health(kind text, resource uuid, new_health text)
RETURNS void AS $$
DECLARE
    state public.notification_health_states%ROWTYPE;
    episode uuid;
    observed timestamptz := clock_timestamp();
BEGIN
    INSERT INTO public.notification_health_states(resource_type, resource_id, generation, health)
    VALUES (kind, resource, pg_catalog.gen_random_uuid(), '') ON CONFLICT DO NOTHING;
    SELECT * INTO state FROM public.notification_health_states
    WHERE resource_type = kind AND resource_id = resource FOR UPDATE;
    new_health := COALESCE(NULLIF(new_health, ''), 'unknown');
    IF state.health = new_health THEN RETURN; END IF;
    episode := state.episode_id;
    IF new_health IN ('warning', 'unhealthy', 'degraded') AND
        (episode IS NULL OR EXISTS (SELECT 1 FROM public.notification_health_episodes WHERE id = episode AND healthy_at IS NOT NULL)) THEN
        episode := pg_catalog.gen_random_uuid();
        INSERT INTO public.notification_health_episodes(id, resource_type, resource_id, started_at)
        VALUES (episode, kind, resource, observed);
    END IF;
    IF new_health = 'healthy' THEN
        UPDATE public.notification_health_episodes SET healthy_at = observed WHERE id = episode AND healthy_at IS NULL;
    END IF;
    UPDATE public.notification_health_states SET health = new_health, generation = pg_catalog.gen_random_uuid(),
        episode_id = episode, healthy_since = CASE WHEN new_health = 'healthy' THEN observed ELSE NULL END
    WHERE resource_type = kind AND resource_id = resource;
END
$$ LANGUAGE plpgsql SET search_path = pg_catalog;

CREATE OR REPLACE FUNCTION notification_health_source_trigger() RETURNS trigger AS $$
DECLARE
    row_data jsonb;
    kind text := TG_ARGV[0];
    resource uuid;
    health text;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF kind = 'check' THEN
            IF OLD.status IS NOT DISTINCT FROM NEW.status THEN RETURN NULL; END IF;
        ELSIF OLD.health IS NOT DISTINCT FROM NEW.health AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at THEN
            RETURN NULL;
        END IF;
    END IF;
    row_data := CASE WHEN TG_OP = 'DELETE' THEN to_jsonb(OLD) ELSE to_jsonb(NEW) END;
    resource := COALESCE(row_data->>'check_id', row_data->>'id')::uuid;
    health := CASE WHEN kind = 'check' THEN row_data->>'status' ELSE row_data->>'health' END;
    IF TG_OP = 'DELETE' OR row_data->>'deleted_at' IS NOT NULL THEN health := 'deleted'; END IF;
    PERFORM public.record_notification_health(kind, resource, health);
    RETURN NULL;
END
$$ LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog;

-- AFTER triggers run alphabetically: stamp the source before its health event is enqueued.
DROP TRIGGER IF EXISTS notification_config_health_marker ON public.config_items;
CREATE OR REPLACE TRIGGER aa_notification_config_health_marker AFTER INSERT OR UPDATE OR DELETE ON public.config_items
FOR EACH ROW EXECUTE FUNCTION notification_health_source_trigger('config');
DROP TRIGGER IF EXISTS notification_component_health_marker ON public.components;
CREATE OR REPLACE TRIGGER aa_notification_component_health_marker AFTER INSERT OR UPDATE OR DELETE ON public.components
FOR EACH ROW EXECUTE FUNCTION notification_health_source_trigger('component');
DROP TRIGGER IF EXISTS notification_check_health_marker ON public.checks_unlogged;
CREATE OR REPLACE TRIGGER aa_notification_check_health_marker AFTER INSERT OR UPDATE OR DELETE ON public.checks_unlogged
FOR EACH ROW EXECUTE FUNCTION notification_health_source_trigger('check');

-- Initializes pre-migration resources lazily and revalidates unlogged check state after restart.
CREATE OR REPLACE FUNCTION refresh_notification_health(kind text, resource uuid) RETURNS void AS $$
DECLARE
    health text;
BEGIN
    CASE kind
    WHEN 'config' THEN
        SELECT CASE WHEN deleted_at IS NULL THEN public.config_items.health::text ELSE 'deleted' END INTO health
        FROM public.config_items WHERE id = resource FOR SHARE;
    WHEN 'component' THEN
        SELECT CASE WHEN deleted_at IS NULL THEN public.components.health::text ELSE 'deleted' END INTO health
        FROM public.components WHERE id = resource FOR SHARE;
    WHEN 'check' THEN
        PERFORM 1 FROM public.checks WHERE id = resource AND deleted_at IS NULL FOR SHARE;
        IF FOUND THEN
            SELECT status INTO health FROM public.checks_unlogged WHERE check_id = resource FOR SHARE;
        END IF;
    ELSE RAISE EXCEPTION 'unsupported recovery resource %', kind;
    END CASE;
    PERFORM public.record_notification_health(kind, resource, COALESCE(health, 'unknown'));
END
$$ LANGUAGE plpgsql SET search_path = pg_catalog;

REVOKE ALL ON FUNCTION public.record_notification_health(text, uuid, text),
    public.refresh_notification_health(text, uuid), public.notification_health_source_trigger() FROM PUBLIC;
