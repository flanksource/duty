-- update_config_item_properties replaces the subset of a config item's
-- properties owned by one creator.
--
-- Ownership model
-- ----------------
-- Each property can be owned by a creator using the metadata fields:
--
--   {
--     "creator_type": "...",
--     "created_by": "..."
--   }
--
-- This function treats (p_creator_type, p_created_by) as the ownership key for
-- the incoming update. It removes existing properties owned by that exact key,
-- preserves everything else, then appends the incoming properties stamped with
-- that same ownership key.
--
-- Preserved properties
-- --------------------
-- The following existing properties are intentionally preserved:
--
--   1. Properties owned by another creator_type / created_by pair.
--   2. Legacy properties that do not have creator_type / created_by metadata.
--   3. Properties whose ownership metadata is incomplete or does not exactly
--      match the incoming ownership key.
--
-- Empty replacement
-- -----------------
-- Passing NULL or [] as p_properties removes this creator's currently-owned
-- properties and appends nothing. Properties owned by others and legacy
-- properties remain untouched.
--
-- Return value
-- ------------
-- Returns one row for the requested config item when it exists:
--
--   changed    true if the row was actually updated, false if the computed
--              properties were identical to the current properties.
--   properties the final/current properties array after the function runs.
--
-- If p_config_id does not match a config_items row, the function returns no
-- rows.
--
-- Why the CTEs exist
-- ------------------
-- stamped:
--   Converts p_properties into an array where every property has creator_type
--   and created_by set to the incoming ownership key.
--
-- computed:
--   Builds the exact final properties array once:
--
--     existing properties not owned by this creator
--     ||
--     incoming stamped properties
--
-- updated:
--   Performs the UPDATE only when the computed value is actually distinct from
--   the current value. This avoids no-op updates, extra row versions, triggers,
--   and misleading changed=true results.
--
-- Example
-- -------
-- Given existing properties:
--
--   [
--     {"name": "legacy"},
--     {"name": "cpu", "value": "old", "creator_type": "scraper", "created_by": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
--     {"name": "region", "value": "us-east", "creator_type": "manual", "created_by": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}
--   ]
--
-- Calling:
--
--   SELECT *
--   FROM update_config_item_properties(
--     '<config-id>',
--     'scraper',
--     'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
--     '[{"name": "cpu", "value": "new"}, {"name": "memory", "value": "high"}]'::jsonb
--   );
--
-- Produces final properties like:
--
--   [
--     {"name": "legacy"},
--     {"name": "region", "value": "us-east", "creator_type": "manual", "created_by": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
--     {"name": "cpu", "value": "new", "creator_type": "scraper", "created_by": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
--     {"name": "memory", "value": "high", "creator_type": "scraper", "created_by": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}
--   ]
--
-- Notes
-- -----
-- Existing preserved properties keep their original relative order. Incoming
-- properties are appended after preserved properties.
--
-- p_properties is expected to be a JSON array. Passing a JSON object, string,
-- number, or boolean will raise an error from jsonb_array_elements().
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
        ORDER BY ord
      ),
      '[]'::jsonb
    ) AS incoming
    FROM jsonb_array_elements(COALESCE(p_properties, '[]'::jsonb))
      WITH ORDINALITY AS incoming(prop, ord)
  ), locked AS (
    SELECT
      ci.id,
      COALESCE(ci.properties, '[]'::jsonb) AS current_properties
    FROM config_items ci
    WHERE ci.id = p_config_id
    FOR UPDATE
  ), computed AS (
    SELECT
      locked.id,
      locked.current_properties,
      COALESCE(
        (
          SELECT jsonb_agg(prop ORDER BY ord)
          FROM jsonb_array_elements(locked.current_properties)
            WITH ORDINALITY AS existing(prop, ord)
          WHERE (
            prop->>'creator_type' = p_creator_type
            AND prop->>'created_by' = p_created_by::text
          ) IS NOT TRUE
        ),
        '[]'::jsonb
      ) || stamped.incoming AS new_properties
    FROM locked
    CROSS JOIN stamped
  ), updated AS (
    UPDATE config_items ci
    SET properties = computed.new_properties
    FROM computed
    WHERE ci.id = computed.id
      AND computed.current_properties IS DISTINCT FROM computed.new_properties
    RETURNING true AS changed, ci.properties
  )
  SELECT updated.changed, updated.properties
  FROM updated
  UNION ALL
  SELECT false AS changed, computed.current_properties AS properties
  FROM computed
  WHERE NOT EXISTS (SELECT 1 FROM updated);
END;
$$ LANGUAGE plpgsql;


-- Deletes a single property by name from the config item's properties owned by
-- (p_creator_type, p_created_by). Properties owned by other creators, and legacy
-- properties without ownership metadata, are preserved.
CREATE
OR REPLACE FUNCTION delete_config_item_property(
  p_config_id uuid,
  p_creator_type TEXT,
  p_created_by uuid,
  p_property_name TEXT
) RETURNS TABLE(changed BOOLEAN, properties jsonb) AS $$
BEGIN
  RETURN QUERY
  WITH updated AS (
    UPDATE config_items ci
    SET properties =
      COALESCE(
        (
          SELECT jsonb_agg(prop ORDER BY ord)
          FROM jsonb_array_elements(COALESCE(ci.properties, '[]'::jsonb))
            WITH ORDINALITY AS existing(prop, ord)
          WHERE (
            prop->>'creator_type' = p_creator_type
            AND prop->>'created_by' = p_created_by::text
            AND prop->>'name' = p_property_name
          ) IS NOT TRUE
        ),
        '[]'::jsonb
      )
    WHERE ci.id = p_config_id
      AND ci.properties IS DISTINCT FROM
      COALESCE(
        (
          SELECT jsonb_agg(prop ORDER BY ord)
          FROM jsonb_array_elements(COALESCE(ci.properties, '[]'::jsonb))
            WITH ORDINALITY AS existing(prop, ord)
          WHERE (
            prop->>'creator_type' = p_creator_type
            AND prop->>'created_by' = p_created_by::text
            AND prop->>'name' = p_property_name
          ) IS NOT TRUE
        ),
        '[]'::jsonb
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