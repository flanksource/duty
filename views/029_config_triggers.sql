CREATE OR REPLACE FUNCTION populate_config_item_name ()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF NEW.name IS NULL OR NEW.NAME = '' THEN
    NEW.name = RIGHT (NEW.id::text, 12);
  END IF;
  
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_config_item_name
  BEFORE INSERT ON config_items
  FOR EACH ROW
  EXECUTE PROCEDURE populate_config_item_name ();

-- Insert config health updates as config changes
CREATE OR REPLACE FUNCTION insert_config_health_updates_as_config_changes ()
  RETURNS TRIGGER
  AS $$
DECLARE
  change_type text;
  severity text := 'info';
  summary text := '';
BEGIN
  -- If record belongs to agent, we ignore it
  IF NEW.agent_id != '00000000-0000-0000-0000-000000000000' THEN
    RETURN NEW;
  END IF;

  IF OLD.health = NEW.health OR (OLD.health IS NULL AND NEW.health IS NULL) OR (OLD IS NULL AND NEW.health = 'unknown') THEN
    RETURN NULL;
  END IF;

  IF NEW.health = 'unknown' OR NEW.health = '' THEN
    change_type := 'HealthUnknown';
  ELSE
    change_type := initcap(NEW.health);
  END IF;

  CASE NEW.health
    WHEN 'unhealthy' THEN severity := 'medium';
    WHEN 'warning' THEN severity := 'low';
    ELSE severity := 'info';
  END CASE;
  
  IF NEW.status IS NOT NULL THEN
    summary := NEW.status;
  END IF;
  
  IF NEW.description IS NOT NULL THEN
    IF summary != '' THEN
      summary := summary || ': ';
    END IF;

    summary := summary || NEW.description;
  END IF;

  INSERT INTO config_changes (config_id, change_type, source, count, severity, summary, details) VALUES (
    NEW.id,
    change_type,
    'config-db',
    1,
    severity,
    summary,
    jsonb_build_object(
      'previous', jsonb_build_object(
        'status', OLD.status,
        'ready', OLD.ready,
        'description', OLD.description
      ),
      'current', jsonb_build_object(
        'status', NEW.status,
        'ready', NEW.ready,
        'description', NEW.description
      )
    )
  );

  RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_health_as_config_changes
  AFTER INSERT OR UPDATE ON config_items
  FOR EACH ROW
  EXECUTE PROCEDURE insert_config_health_updates_as_config_changes();

-- Normalize external_users aliases: lowercase, deduplicate, and sort
CREATE OR REPLACE FUNCTION normalize_external_users_aliases()
  RETURNS TRIGGER
  AS $$
BEGIN
  IF NEW.aliases IS NOT NULL THEN
    NEW.aliases := ARRAY(SELECT DISTINCT LOWER(elem) FROM unnest(NEW.aliases) AS elem ORDER BY LOWER(elem));
  END IF;
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER normalize_external_users_aliases_trigger
  BEFORE INSERT OR UPDATE ON external_users
  FOR EACH ROW
  EXECUTE PROCEDURE normalize_external_users_aliases();
