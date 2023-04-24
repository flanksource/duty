-- Insert identities in people table
CREATE
OR REPLACE FUNCTION insert_identity_to_people () RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO people (id, name, email)
    VALUES (NEW.id, concat(NEW.traits::json->'name'->>'first', ' ', NEW.traits::json->'name'->>'last'), NEW.traits::json->>'email');

    -- Give any new user viewer role
    INSERT INTO casbin_rule (ptype, v0, v1)
    VALUES ('g', NEW.id::text, 'viewer');

    RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'identities') THEN
        CREATE OR REPLACE TRIGGER identity_to_people
            AFTER INSERT
            ON identities
            FOR EACH ROW
            EXECUTE PROCEDURE insert_identity_to_people();
    END IF;
END $$;
