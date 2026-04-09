-- dependsOn: functions/drop.sql, views/030_config_changes.sql

-- Maintains change_groups.member_count, last_member_at, ended_at and started_at
-- when a config_changes row is attached to a group via group_id.
-- Runs on INSERT and on UPDATE OF group_id only, so the dedup UPDATE path in
-- config_changes_update_trigger() does not re-evaluate membership.
CREATE OR REPLACE FUNCTION change_groups_maintain_members()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.group_id IS NULL THEN
    RETURN NEW;
  END IF;

  IF TG_OP = 'UPDATE' AND OLD.group_id IS NOT DISTINCT FROM NEW.group_id THEN
    RETURN NEW;
  END IF;

  UPDATE change_groups g
  SET
    member_count   = g.member_count + 1,
    last_member_at = GREATEST(g.last_member_at, NEW.created_at),
    ended_at       = CASE
                       WHEN g.ended_at IS NULL THEN NULL
                       ELSE GREATEST(g.ended_at, NEW.created_at)
                     END,
    started_at     = CASE
                       WHEN g.member_count = 0 THEN NEW.created_at
                       ELSE LEAST(g.started_at, NEW.created_at)
                     END,
    updated_at     = now()
  WHERE g.id = NEW.group_id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_change_groups_maintain_members_ins ON config_changes;
CREATE TRIGGER trg_change_groups_maintain_members_ins
AFTER INSERT ON config_changes
FOR EACH ROW
EXECUTE FUNCTION change_groups_maintain_members();

DROP TRIGGER IF EXISTS trg_change_groups_maintain_members_upd ON config_changes;
CREATE TRIGGER trg_change_groups_maintain_members_upd
AFTER UPDATE OF group_id ON config_changes
FOR EACH ROW
EXECUTE FUNCTION change_groups_maintain_members();

-- Aggregated view used by the query layer / UI.
CREATE OR REPLACE VIEW change_groups_summary AS
SELECT
  g.id,
  g.type,
  g.summary,
  g.source,
  g.rule_name,
  g.status,
  g.started_at,
  g.ended_at,
  g.last_member_at,
  g.member_count,
  COUNT(DISTINCT cc.config_id)                                                 AS distinct_config_count,
  EXTRACT(EPOCH FROM (COALESCE(g.ended_at, g.last_member_at) - g.started_at))  AS duration_seconds,
  g.details,
  g.created_at,
  g.updated_at
FROM change_groups g
LEFT JOIN config_changes cc ON cc.group_id = g.id
GROUP BY g.id;
