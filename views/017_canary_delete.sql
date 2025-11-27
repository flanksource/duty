CREATE OR REPLACE FUNCTION soft_delete_canary ()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE check_component_relationships
  SET deleted_at = NEW.deleted_at
  WHERE canary_id = OLD.id AND deleted_at IS NULL;

  UPDATE checks
  SET deleted_at = NEW.deleted_at
  WHERE canary_id = OLD.id AND deleted_at IS NULL;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER canary_deleted_trigger
AFTER UPDATE ON canaries
FOR EACH ROW
WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
EXECUTE FUNCTION soft_delete_canary();

CREATE OR REPLACE FUNCTION soft_delete_check ()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE check_component_relationships
  SET deleted_at = NEW.deleted_at
  WHERE check_id = OLD.id AND deleted_at IS NULL;

  UPDATE canaries
  SET deleted_at = NOW()
  WHERE
    deleted_at IS NULL AND
    source = 'check=' || OLD.id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_deleted_trigger
AFTER UPDATE ON checks
FOR EACH ROW
WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
EXECUTE FUNCTION soft_delete_check();
