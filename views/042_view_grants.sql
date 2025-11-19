-- Function to check if user has access to a row based on grants
-- Grants are a jsonb array of scope UUIDs that are allowed to access the row
-- User scopes are provided in JWT claims
-- NULL or empty grants means row is hidden from all (no access)
--
-- Examples:
--   User scopes: ['scope-a', 'scope-b'], Row grants: ['scope-a', 'scope-c'] → TRUE (scope-a overlaps)
--   User scopes: ['scope-x'], Row grants: ['scope-a', 'scope-b'] → FALSE (no overlap)
--   User scopes: ['scope-a'], Row grants: NULL → FALSE (hidden)
--   User scopes: ['scope-a'], Row grants: [] → FALSE (hidden)
CREATE OR REPLACE FUNCTION check_view_grants(grants jsonb) RETURNS BOOLEAN AS $$
BEGIN
  -- NULL or empty array means row is hidden from all
  IF grants IS NULL OR jsonb_array_length(grants) = 0 THEN
    RETURN FALSE;
  END IF;

  -- Check if any user scope UUID exists in grants array
  RETURN EXISTS (
    SELECT 1 FROM jsonb_array_elements_text(grants) AS grant_uuid
    WHERE grant_uuid = ANY(
      COALESCE(
        ARRAY(SELECT jsonb_array_elements_text(
          current_setting('request.jwt.claims', TRUE)::jsonb -> 'scopes'
        )), '{}'::text[]
      )
    )
  );
END;
$$ LANGUAGE plpgsql STABLE;
