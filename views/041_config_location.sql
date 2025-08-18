-- Returns ids of config items that are children of the given config item based on location.
CREATE OR REPLACE FUNCTION find_children_by_location(config_id uuid, include_deleted boolean DEFAULT false)
RETURNS TABLE(id uuid, type text, name text)
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ci.id,
        ci.type,
        ci.name
    FROM config_items ci
    WHERE 
        EXISTS (
            SELECT 1 
            FROM config_items parent
            CROSS JOIN unnest(parent.aliases) AS parent_alias
            CROSS JOIN unnest(ci.locations) AS loc
            WHERE parent.id = config_id
            AND loc LIKE parent_alias || '%' 
        )
        AND ci.id != config_id
        AND (include_deleted = true OR ci.deleted_at IS NULL);
END
$$
LANGUAGE plpgsql
STABLE;