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