-- An access token is limited by the RoleBindings that select it, not by Roles stored on it.
-- Runs ahead of the schema apply, which no longer has access_tokens.roles.
--
-- Each Role a person's token lists becomes a RoleBinding in the reserved mission-control-access-tokens namespace
-- whose only subject is the token's id. Its name and shape match the bindings Mission Control creates with a token.
-- A person's token that lists no Role was refused anyway, so it's deleted. Agents' tokens list none, and are kept.
DO $$
DECLARE
  token RECORD;
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'access_tokens' AND column_name = 'roles'
  ) THEN
    INSERT INTO role_bindings (name, namespace, description, role, subjects, source, created_by)
    SELECT
      'token-' || t.id || '-' || left(md5((role ->> 'namespace') || '/' || (role ->> 'name')), 12),
      'mission-control-access-tokens',
      'Limits access token ' || t.name,
      (role ->> 'namespace') || '/' || (role ->> 'name'),
      jsonb_build_object('accessToken', t.id::text),
      'AccessToken',
      t.person_id
    FROM access_tokens t
    JOIN people p ON p.id = t.person_id
    CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(t.roles) = 'array' THEN t.roles ELSE '[]'::jsonb END) AS role
    WHERE p.type IS DISTINCT FROM 'agent'
      AND COALESCE(role ->> 'namespace', '') <> ''
      AND COALESCE(role ->> 'name', '') <> ''
    ON CONFLICT DO NOTHING;

    FOR token IN
      SELECT t.id, t.name, p.email FROM access_tokens t LEFT JOIN people p ON p.id = t.person_id
      WHERE p.type IS DISTINCT FROM 'agent'
        AND COALESCE(jsonb_typeof(t.roles) <> 'array' OR t.roles = '[]'::jsonb, TRUE)
    LOOP
      RAISE NOTICE 'deleting access token % of % (%): it lists no role', token.name, token.email, token.id;
      DELETE FROM access_tokens WHERE id = token.id;
    END LOOP;

    ALTER TABLE access_tokens DROP COLUMN roles;
  END IF;
END $$;
