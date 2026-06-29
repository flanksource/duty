-- Notify on any updates on the teams table
CREATE OR REPLACE FUNCTION handle_team_updates()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
    DELETE FROM team_components WHERE team_id = OLD.id;
  END IF;

  IF OLD.spec != NEW.spec THEN
    UPDATE notifications SET error = NULL WHERE team_id = NEW.id;
  END IF;

  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER team_updates
AFTER UPDATE ON teams
FOR EACH ROW EXECUTE PROCEDURE handle_team_updates();

CREATE OR REPLACE FUNCTION handle_person_analytics_insert()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_advisory_xact_lock(hashtextextended(NEW.person_id::text || ':' || NEW.key, 0));

  UPDATE person_analytics
  SET
    updated_at = clock_timestamp(),
    count = count + 1
  WHERE person_id = NEW.person_id
    AND key = NEW.key;

  IF FOUND THEN
    RETURN NULL;
  END IF;

  NEW.updated_at = COALESCE(NEW.updated_at, clock_timestamp());
  NEW.count = COALESCE(NEW.count, 1);
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER person_analytics_insert
BEFORE INSERT ON person_analytics
FOR EACH ROW EXECUTE PROCEDURE handle_person_analytics_insert();
