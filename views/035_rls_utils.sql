-- rls_scope_access returns scope UUIDs from request.jwt.claims (empty when missing).
CREATE
OR REPLACE FUNCTION rls_scope_access() RETURNS UUID[] AS $$
DECLARE
  jwt_claims TEXT;
BEGIN
  jwt_claims := current_setting('request.jwt.claims', TRUE);
  IF jwt_claims IS NULL OR jwt_claims = '' THEN
    RETURN '{}'::uuid[];
  END IF;

  RETURN COALESCE(
    ARRAY(SELECT jsonb_array_elements_text(jwt_claims::jsonb -> 'scopes')::uuid),
    '{}'::uuid[]
  );
END;
$$ LANGUAGE plpgsql STABLE SECURITY INVOKER;
