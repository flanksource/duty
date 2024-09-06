CREATE OR REPLACE FUNCTION update_config_change(
  increment_count integer,
  p_id uuid,
  p_change_type TEXT,
  p_count INTEGER,
  p_created_by uuid,
  p_details jsonb,
  p_diff TEXT,
  p_external_change_id TEXT,
  p_external_created_by TEXT,
  p_patches jsonb,
  p_severity TEXT,
  p_source TEXT,
  p_summary TEXT
)
RETURNS void
AS $$
DECLARE current_details jsonb;
BEGIN
  SELECT details INTO current_details FROM config_changes WHERE id = p_id;

  UPDATE config_changes
  SET
    change_type = p_change_type,
    count = CASE
      WHEN current_details IS DISTINCT FROM p_details THEN count + increment_count
      ELSE count
    END,
    created_at = NOW(),
    created_by = p_created_by,
    details = p_details,
    diff = p_diff,
    external_change_id = p_external_change_id,
    external_created_by = p_external_created_by,
    patches = p_patches,
    severity = p_severity,
    source = p_source,
    summary = p_summary
  WHERE 
    id = p_id;
  EXCEPTION
    WHEN unique_violation THEN
      IF sqlerrm LIKE '%config_changes_config_id_external_change_id_key%' THEN
        UPDATE config_changes
        SET
          change_type = p_change_type,
          count = CASE
            WHEN current_details IS DISTINCT FROM p_details THEN count + increment_count
            ELSE count
          END,
          created_at = NOW(),
          created_by = p_created_by,
          details = p_details,
          diff = p_diff,
          external_created_by = p_external_created_by,
          patches = p_patches,
          severity = p_severity,
          source = p_source,
          summary = p_summary
        WHERE 
          external_change_id = p_external_change_id;
      ELSE
        RAISE;
      END IF;
    WHEN OTHERS THEN
      RAISE;
END;
$$ LANGUAGE plpgsql;