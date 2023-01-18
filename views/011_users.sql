CREATE TABLE IF NOT EXISTS identities();

-- Insert identities in people table
CREATE OR REPLACE FUNCTION insert_identity_to_people()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO people(id, name, email)
    VALUES (NEW.id, concat(NEW.traits::json->'name'->>'first', ' ', NEW.traits::json->'name'->>'last'), NEW.traits::json->>'email');
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER identity_to_people
    AFTER INSERT
    ON identities
    FOR EACH ROW
    EXECUTE PROCEDURE insert_identity_to_people();


