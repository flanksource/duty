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
