-- isolated from 9998_rls_enable.sql because generated tables in the view use it.
CREATE
OR REPLACE FUNCTION is_rls_disabled() RETURNS BOOLEAN AS $$
DECLARE
  jwt_claims TEXT;
BEGIN
  jwt_claims := current_setting('request.jwt.claims', TRUE);
  RETURN (jwt_claims IS NULL
    OR jwt_claims = ''
    OR jwt_claims::jsonb ->> 'disable_rls' IS NOT NULL);
END;
$$ LANGUAGE plpgsql SECURITY INVOKER;

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

-- rls_has_wildcard reports whether request.jwt.claims includes the given wildcard scope type.
CREATE
OR REPLACE FUNCTION rls_has_wildcard(scope_type TEXT) RETURNS BOOLEAN AS $$
DECLARE
  jwt_claims TEXT;
BEGIN
  jwt_claims := current_setting('request.jwt.claims', TRUE);
  IF jwt_claims IS NULL OR jwt_claims = '' THEN
    RETURN FALSE;
  END IF;

  RETURN EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(
      COALESCE(jwt_claims::jsonb -> 'wildcard_scopes', '[]'::jsonb)
    ) AS wildcard
    WHERE wildcard = scope_type
  );
END;
$$ LANGUAGE plpgsql STABLE SECURITY INVOKER;
