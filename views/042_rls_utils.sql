-- isolated from 9998_rls_enable.sql because generated tables in the view use it.
CREATE
OR REPLACE FUNCTION is_rls_disabled () RETURNS BOOLEAN AS $$
BEGIN
  RETURN (current_setting('request.jwt.claims', TRUE) IS NULL
    OR current_setting('request.jwt.claims', TRUE) = ''
    OR current_setting('request.jwt.claims', TRUE)::jsonb ->> 'disable_rls' IS NOT NULL);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;