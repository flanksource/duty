CREATE OR REPLACE FUNCTION config_changes_update_trigger() 
RETURNS TRIGGER AS $$
DECLARE 
  count_increment INT;
BEGIN
  count_increment := NEW.count - OLD.count;

  UPDATE
    config_changes
  SET
    change_type = NEW.change_type,
    count = CASE
      WHEN NEW.details IS DISTINCT FROM OLD.details OR NEW.diff IS DISTINCT FROM OLD.diff THEN NEW.count
      ELSE count
    END,
    created_at = CASE
      WHEN NEW.details IS DISTINCT FROM OLD.details OR NEW.diff IS DISTINCT FROM OLD.diff THEN NEW.created_at
      ELSE OLD.created_at
    END,
    created_by = NEW.created_by,
    details = NEW.details,
    is_pushed = NEW.is_pushed,
    diff = NEW.diff,
    external_created_by = NEW.external_created_by,
    external_change_id = NEW.external_change_id,
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
          WHEN NEW.details IS DISTINCT FROM OLD.details OR NEW.diff IS DISTINCT FROM OLD.diff THEN config_changes.count + count_increment
          ELSE count
        END,
        created_at = CASE
          WHEN NEW.details IS DISTINCT FROM OLD.details OR NEW.diff IS DISTINCT FROM OLD.diff THEN NOW()
          ELSE COALESCE(NEW.created_at, OLD.created_at)
        END,
        created_by = NEW.created_by,
        details = NEW.details,
        diff = NEW.diff,
        external_created_by = NEW.external_created_by,
        patches = NEW.patches,
        severity = NEW.severity,
        source = NEW.source,
        summary = NEW.summary
      WHERE 
        external_change_id = NEW.external_change_id AND config_id = NEW.config_id;

      RETURN NULL;
    ELSE
      RAISE;
    END IF;
  WHEN OTHERS THEN
    RAISE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_changes_update_trigger
BEFORE UPDATE
ON config_changes FOR EACH ROW
WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION config_changes_update_trigger();

---
CREATE OR REPLACE FUNCTION config_changes_insert_trigger() 
RETURNS TRIGGER AS $$
DECLARE
  existing_details JSONB;
  existing_created_at TIMESTAMP WITH TIME ZONE;
BEGIN
  -- run the original insert manually.
  INSERT INTO config_changes SELECT NEW.* 
  ON CONFLICT (id) 
  DO UPDATE 
  SET 
    details = excluded.details,
    created_by = excluded.created_by,
    diff = excluded.diff,
    external_created_by = excluded.external_created_by,
    patches = excluded.patches,
    severity = excluded.severity,
    fingerprint = excluded.fingerprint,
    count = excluded.count,
    source = excluded.source,
    created_at = excluded.created_at,
    summary = excluded.summary;
    
  -- Prevent the original insert by returning NULL
  RETURN NULL;
EXCEPTION
  WHEN unique_violation THEN
    IF sqlerrm LIKE '%config_changes_config_id_external_change_id_key%' THEN
      SELECT details, created_at FROM config_changes 
      WHERE external_change_id = NEW.external_change_id AND config_id = NEW.config_id
      INTO existing_details, existing_created_at;

      UPDATE config_changes
      SET
        change_type = NEW.change_type,
        count = CASE
          WHEN (NEW.details IS DISTINCT FROM existing_details OR NEW.created_at IS DISTINCT FROM existing_created_at) THEN config_changes.count + 1
          ELSE count
        END,
        created_at = NEW.created_at,
        created_by = NEW.created_by,
        details = NEW.details,
        diff = NEW.diff,
        external_created_by = NEW.external_created_by,
        patches = NEW.patches,
        severity = NEW.severity,
        source = NEW.source,
        summary = NEW.summary
      WHERE 
        external_change_id = NEW.external_change_id
        AND config_id = NEW.config_id;

      RETURN NULL;
    ELSE
      RAISE;
    END IF;
  WHEN OTHERS THEN
    RAISE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_changes_insert_trigger
BEFORE INSERT
ON config_changes FOR EACH ROW
WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION config_changes_insert_trigger();
---