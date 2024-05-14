-- Push hard deletes for tables that we sync to upstream
CREATE
OR REPLACE FUNCTION push_deletes_to_event_queue () RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
    payload JSONB;
    priority integer := 0;
    priority_table JSONB := '{
        "topologies": 20,
        "canaries": 20,
        "config_scrapers": 20,
        "checks": 40,
        "components": 40,
        "config_items": 40,
        "config_component_relationships": 50,
        "component_relationships": 50,
        "config_relationships": 50
    }';
BEGIN
  IF TG_OP != 'DELETE' THEN
    RETURN NULL;
  END IF;

  rec = OLD;
  CASE TG_TABLE_NAME
    WHEN 'component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'relationship_id', rec.relationship_id, 'selector_id', rec.selector_id);
    WHEN 'config_component_relationships' THEN
      payload = jsonb_build_object('component_id', rec.component_id, 'config_id', rec.config_id);
    WHEN 'config_relationships' THEN
      payload = jsonb_build_object('related_id', rec.related_id, 'config_id', rec.config_id, 'relation', rec.relation);
    WHEN 'check_statuses' THEN
       RETURN NULL;
    ELSE
      payload = jsonb_build_object('id', rec.id);
  END CASE;

  -- Log changes to event queue
  priority = (priority_table->>TG_TABLE_NAME)::integer;
  INSERT INTO event_queue (name, properties, priority) VALUES ('push_queue.delete', jsonb_build_object('table', TG_TABLE_NAME) || payload, priority)
  ON CONFLICT (name, properties) DO UPDATE SET created_at = NOW(), last_attempt = NULL, attempts = 0;

  RETURN NULL;
END;
$$ LANGUAGE 'plpgsql' SECURITY DEFINER;

-- Apply trigger
DO $$
DECLARE
  table_name TEXT;
BEGIN
  FOR table_name IN
    SELECT t.table_name
    FROM information_schema.tables  t
    WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
      AND t.table_name IN (
        'topologies',
        'canaries',
        'config_scrapers',
        'checks',
        'components',
        'config_items',
        'config_component_relationships',
        'component_relationships',
        'config_relationships'
      )
  LOOP
    EXECUTE format('
      CREATE OR REPLACE TRIGGER %1$I_change_to_event_queue
      AFTER DELETE ON %1$I
      FOR EACH ROW
      EXECUTE PROCEDURE push_deletes_to_event_queue()',
      table_name
    );
  END LOOP;
END $$;
