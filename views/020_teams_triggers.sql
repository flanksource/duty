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

-- Agents can't be in a team: a membership would give an agent the team's grants.
CREATE OR REPLACE FUNCTION team_members_refuse_agents()
RETURNS TRIGGER AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM people WHERE id = NEW.person_id AND type = 'agent') THEN
    RAISE EXCEPTION 'person % is an agent, and agents cannot be team members', NEW.person_id
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER team_members_refuse_agents
BEFORE INSERT OR UPDATE ON team_members
FOR EACH ROW EXECUTE PROCEDURE team_members_refuse_agents();

-- A person can't become an agent or stop being one: either change would bypass the refusal above.
CREATE OR REPLACE FUNCTION people_agent_type_immutable()
RETURNS TRIGGER AS $$
BEGIN
  IF (OLD.type IS NOT DISTINCT FROM 'agent') <> (NEW.type IS NOT DISTINCT FROM 'agent') THEN
    RAISE EXCEPTION 'person % cannot change its type to or from agent', NEW.id
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER people_agent_type_immutable
BEFORE UPDATE OF type ON people
FOR EACH ROW EXECUTE PROCEDURE people_agent_type_immutable();

DELETE FROM casbin_rule
WHERE ptype = 'g'
  AND v0 IN (SELECT id::TEXT FROM people WHERE type = 'agent')
  AND v1 IN (SELECT id::TEXT FROM teams);

DELETE FROM team_members tm USING people p WHERE p.id = tm.person_id AND p.type = 'agent';
