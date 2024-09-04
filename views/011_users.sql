-- Insert identities in people table
CREATE OR REPLACE FUNCTION sync_identity_to_people () 
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    INSERT INTO people (id, name, email)
    VALUES (NEW.id, concat(NEW.traits::json->'name'->>'first', ' ', NEW.traits::json->'name'->>'last'), NEW.traits::json->>'email');
  ELSIF TG_OP = 'UPDATE' THEN
    UPDATE people SET
      name = concat(NEW.traits::json->'name'->>'first', ' ', NEW.traits::json->'name'->>'last'),
      email = NEW.traits::json->>'email'
      WHERE id = NEW.id;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'identities') THEN
    CREATE OR REPLACE TRIGGER identity_to_people
    AFTER INSERT OR UPDATE ON identities
    FOR EACH ROW
    EXECUTE PROCEDURE sync_identity_to_people();
  END IF;
END $$;

CREATE OR REPLACE VIEW
  people_roles AS
SELECT
  people.id,
  people.name,
  people.email,
  array_agg(cr.v1) AS roles
FROM
  people
  INNER JOIN casbin_rule cr ON cr.v0 = people.id::VARCHAR
GROUP BY
  people.id;
