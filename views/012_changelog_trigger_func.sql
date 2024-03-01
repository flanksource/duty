-- Push table changes to event queue
CREATE
OR REPLACE FUNCTION push_changes_to_event_queue () RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
    payload JSONB;
    event_name TEXT := 'push_queue.create';
    priority integer := 0;
    priority_table JSONB := '{
        "topologies": 20,
        "canaries": 20,
        "config_scrapers": 20,
        "checks": 10,
        "components": 10,
        "config_items": 10,
        "config_component_relationships": 5,
        "component_relationships": 5,
        "config_relationships": 5
    }';
BEGIN
  rec = NEW;
  
  IF TG_OP = 'DELETE' THEN
    rec = OLD;
    event_name = 'push_queue.delete';
  END IF;

  CASE TG_TABLE_NAME
    WHEN 'component_relationships' THEN
      IF TG_OP != 'DELETE' THEN
        -- Set these fields to null for component_relationships to prevent excessive pushes
        rec.updated_at = NULL;
        OLD.updated_at = NULL;

        -- If it is same as the old record, then no action required
        IF rec IS NOT DISTINCT FROM OLD THEN
          RETURN NULL;
        END IF;
      END IF;

      payload = jsonb_build_object('component_id', rec.component_id, 'relationship_id', rec.relationship_id, 'selector_id', rec.selector_id);
    WHEN 'config_component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'config_id', rec.config_id);
    WHEN 'config_relationships' THEN
      payload = jsonb_build_object('related_id', rec.related_id, 'config_id', rec.config_id, 'selector_id', rec.selector_id);
    WHEN 'checks' THEN
      IF TG_OP != 'DELETE' THEN
        -- Set these fields to null for checks to prevent excessive pushes
        rec.updated_at = NULL;
        OLD.updated_at = NULL;

        -- If it is same as the old record, then no action required
        IF rec IS NOT DISTINCT FROM OLD THEN
          RETURN NULL;
        END IF;
      END IF;

      payload = jsonb_build_object('id', rec.id);
    WHEN 'canaries' THEN
      IF TG_OP != 'DELETE' THEN
        -- Set these fields to null for canaries to prevent excessive pushes
        rec.updated_at = NULL;
        OLD.updated_at = NULL;

        -- If it is same as the old record, then no action required
        IF rec IS NOT DISTINCT FROM OLD THEN
          RETURN NULL;
        END IF;
      END IF;

      payload = jsonb_build_object('id', rec.id);
    ELSE
      payload = jsonb_build_object('id', rec.id);
  END CASE;

  -- Log changes to event queue
  priority = (priority_table->>TG_TABLE_NAME)::integer;
  INSERT INTO event_queue (name, properties, priority) VALUES (event_name, jsonb_build_object('table', TG_TABLE_NAME) || payload, priority)
  ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;

  RETURN NULL;
END;
$$ LANGUAGE 'plpgsql' SECURITY DEFINER;