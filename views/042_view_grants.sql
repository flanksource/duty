-- Function to check if user has access to a row based on grants
-- Grants are a jsonb array of scope UUIDs that are allowed to access the row
-- User scopes are provided in JWT claims
CREATE OR REPLACE FUNCTION check_view_grants(grants jsonb) RETURNS BOOLEAN AS $$
DECLARE
    grants_array text[];
    user_scopes text[];
BEGIN
    -- NULL grants means no RLS enforcement (bypass)
    IF grants IS NULL THEN
        RETURN TRUE;
    END IF;

    -- Empty array means row is hidden from all
    IF jsonb_array_length(grants) = 0 THEN
        RETURN FALSE;
    END IF;

    -- Convert jsonb array of scope UUIDs to text array
    grants_array := ARRAY(SELECT jsonb_array_elements_text(grants));

    -- Get user scopes from JWT claims
    user_scopes := COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(
            current_setting('request.jwt.claims', TRUE)::jsonb -> 'scopes'
        )), '{}'::text[]
    );

    -- Check if user has any of the granted scopes
    RETURN grants_array && user_scopes;
END;
$$ LANGUAGE plpgsql STABLE;
