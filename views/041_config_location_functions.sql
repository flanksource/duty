---
CREATE INDEX IF NOT EXISTS config_locations_location_pattern_idx ON config_locations (location text_pattern_ops);

-- Function to get children by location based on config external IDs
CREATE OR REPLACE FUNCTION get_children_by_location(
    config_id_param UUID, 
    location_prefix TEXT DEFAULT NULL,
    include_deleted BOOLEAN DEFAULT FALSE
)
RETURNS TABLE (
    id UUID,
    type TEXT,
    name TEXT
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
    IF location_prefix IS NOT NULL AND location_prefix <> '' THEN
        filtered_external_ids := ARRAY(
            SELECT unnest(filtered_external_ids) AS ext_id
            WHERE ext_id LIKE location_prefix || '%'
        );
    END IF;
    
    -- For each filtered external_id, find configs that have that prefix in their location
    FOREACH ext_id IN ARRAY filtered_external_ids
    LOOP
        RETURN QUERY
        SELECT 
            cl.id,
            ci.type,
            ci.name
        FROM config_locations cl
        JOIN config_items ci ON cl.id = ci.id
        WHERE cl.location LIKE ext_id || '%'
        AND (include_deleted OR ci.deleted_at IS NULL);
    END LOOP;
END;
$$ LANGUAGE plpgsql;