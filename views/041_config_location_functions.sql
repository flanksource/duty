---
CREATE INDEX IF NOT EXISTS config_locations_location_pattern_idx ON config_locations (location text_pattern_ops);

-- Function to get children by location based on config external IDs
CREATE OR REPLACE FUNCTION get_children_by_location(
    config_id_param UUID, 
    alias_prefix TEXT DEFAULT NULL
)
RETURNS TABLE (
    id UUID
) AS $$
DECLARE
    ext_id TEXT;
    filtered_external_ids TEXT[];
BEGIN
    -- Get the config item's external_id array
    SELECT external_id INTO filtered_external_ids 
    FROM config_items 
    WHERE config_items.id = config_id_param;
    
    -- If no external_id found, return empty result
    IF filtered_external_ids IS NULL THEN
        RETURN;
    END IF;
    
    -- Filter external_ids by prefix if provided
    IF alias_prefix IS NOT NULL AND alias_prefix <> '' THEN
        filtered_external_ids := ARRAY(
            SELECT ext_id_val
            FROM unnest(filtered_external_ids) AS ext_id_val
            WHERE ext_id_val = alias_prefix OR ext_id_val LIKE alias_prefix || '%'
        );
    END IF;
    
    -- For each filtered external_id, find configs that have that prefix in their location
    FOREACH ext_id IN ARRAY filtered_external_ids
    LOOP
        RETURN QUERY
        SELECT cl.id
        FROM config_locations cl
        WHERE cl.location LIKE ext_id || '%' OR cl.location = ext_id;
    END LOOP;
END;
$$ LANGUAGE plpgsql;