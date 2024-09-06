CREATE
OR REPLACE FUNCTION config_changes_update_trigger() RETURNS TRIGGER AS $$
DECLARE 
  count_increment INT;
BEGIN
  count_increment := NEW.count - OLD.count;

  UPDATE
    config_changes
  SET
    change_type = NEW.change_type,
    count = CASE
      WHEN NEW.details IS DISTINCT FROM OLD.details THEN NEW.count
      ELSE count
    END,
    created_at = NOW(),
    created_by = NEW.created_by,
    details = NEW.details,
    diff = NEW.diff,
    external_created_by = NEW.external_created_by,
    external_change_id = NEW.external_change_id,
    first_observed = LEAST(first_observed, created_at),
    patches = NEW.patches,
    severity = NEW.severity,
    source = NEW.source,
    summary = NEW.summary
  WHERE
    id = NEW.id;
    
  -- Prevent the original update by returning NULL
  RETURN NULL;
EXCEPTION
  WHEN unique_violation THEN
    IF sqlerrm LIKE '%config_changes_config_id_external_change_id_key%' THEN
      UPDATE config_changes
      SET
        change_type = NEW.change_type,
        count = CASE
          WHEN NEW.details IS DISTINCT FROM OLD.details THEN config_changes.count + count_increment
          ELSE count
        END,
        created_at = NOW(),
        created_by = NEW.created_by,
        details = NEW.details,
        diff = NEW.diff,
        external_created_by = NEW.external_created_by,
        first_observed = LEAST(first_observed, created_at),
        patches = NEW.patches,
        severity = NEW.severity,
        source = NEW.source,
        summary = NEW.summary
      WHERE 
        external_change_id = NEW.external_change_id;

      RETURN NULL;
    ELSE
      RAISE;
    END IF;
  WHEN OTHERS THEN
    RAISE;
END;
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER config_changes_update_trigger BEFORE
UPDATE
  ON config_changes FOR EACH ROW
  WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION config_changes_update_trigger();