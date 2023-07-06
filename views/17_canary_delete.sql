CREATE OR REPLACE FUNCTION update_checks_and_relationships_deleted_at ()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE checks
  SET deleted_at = NEW.deleted_at
  WHERE canary_id = OLD.id;

  UPDATE check_component_relationships
  SET deleted_at = NEW.deleted_at
  WHERE canary_id = OLD.id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER canary_deleted_trigger
AFTER UPDATE ON canaries
FOR EACH ROW
WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
EXECUTE FUNCTION update_checks_and_relationships_deleted_at();