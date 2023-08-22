CREATE OR REPLACE FUNCTION handle_component_updates () 
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
    DELETE FROM team_components WHERE component_id = OLD.id;

    UPDATE component_relationships SET deleted_at = NOW() WHERE component_id = OLD.id OR relationship_id = OLD.id;

    UPDATE check_component_relationships SET deleted_at = NOW() WHERE component_id = OLD.id;

    UPDATE config_component_relationships SET deleted_at = NOW() WHERE component_id = OLD.id;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER component_updates
AFTER UPDATE ON components
FOR EACH ROW EXECUTE PROCEDURE handle_component_updates();