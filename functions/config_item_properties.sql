-- Replaces the config item's properties owned by (p_creator_type, p_created_by)
-- with p_properties stamped with that ownership. Properties owned by other
-- creators, and legacy properties without ownership metadata, are preserved;
-- passing an empty/null p_properties removes this creator's owned properties.
CREATE OR REPLACE FUNCTION update_config_item_properties(
  p_config_id uuid,
  p_creator_type text,
  p_created_by uuid,
  p_properties jsonb
) RETURNS TABLE(changed boolean, properties jsonb) AS $$
BEGIN
  RETURN QUERY
  WITH stamped AS (
    SELECT COALESCE(
      jsonb_agg(
        prop || jsonb_build_object(
          'creator_type', p_creator_type,
          'created_by', p_created_by::text
        )
      ),
      '[]'::jsonb
    ) AS incoming
    FROM jsonb_array_elements(COALESCE(p_properties, '[]'::jsonb)) AS incoming(prop)
  ), updated AS (
    UPDATE config_items ci
    SET properties =
      (
        COALESCE(
          (
            SELECT jsonb_agg(prop ORDER BY ord)
            FROM jsonb_array_elements(COALESCE(ci.properties, '[]'::jsonb))
              WITH ORDINALITY AS existing(prop, ord)
            WHERE (
              prop->>'creator_type' = p_creator_type
              AND prop->>'created_by' = p_created_by::text
            ) IS NOT TRUE
          ),
          '[]'::jsonb
        )
        || (SELECT incoming FROM stamped)
      )
    WHERE ci.id = p_config_id
      AND ci.properties IS DISTINCT FROM
      (
        COALESCE(
          (
            SELECT jsonb_agg(prop ORDER BY ord)
            FROM jsonb_array_elements(COALESCE(ci.properties, '[]'::jsonb))
              WITH ORDINALITY AS existing(prop, ord)
            WHERE (
              prop->>'creator_type' = p_creator_type
              AND prop->>'created_by' = p_created_by::text
            ) IS NOT TRUE
          ),
          '[]'::jsonb
        )
        || (SELECT incoming FROM stamped)
      )
    RETURNING true AS changed, ci.properties
  )
  SELECT updated.changed, updated.properties
  FROM updated
  UNION ALL
  SELECT false AS changed, ci.properties
  FROM config_items ci
  WHERE ci.id = p_config_id
    AND NOT EXISTS (SELECT 1 FROM updated);
END;
$$ LANGUAGE plpgsql;
