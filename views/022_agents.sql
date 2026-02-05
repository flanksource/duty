DROP VIEW IF EXISTS agents_summary;

CREATE OR REPLACE VIEW agents_summary AS
SELECT
    agents.*,
    configs.count AS config_count,
    config_scrapper.count AS config_scrapper_count,
    checks.count AS checks_count,
    playbook_runs.count AS playbook_runs_count,
    CASE
      WHEN last_seen IS NULL THEN 'unknown'
      WHEN EXTRACT(EPOCH FROM (NOW() - last_seen)) < 61 THEN 'healthy'
      ELSE 'unhealthy'
    END AS health,
    CASE
      WHEN last_seen IS NULL THEN 'unknown'
      WHEN EXTRACT(EPOCH FROM (NOW() - last_seen)) < 61 THEN 'online'
      ELSE 'offline'
    END AS status
FROM
    agents
    LEFT JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            config_items
        GROUP BY
            agent_id
    ) AS configs ON configs.agent_id = agents.id
    LEFT JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            config_scrapers
        GROUP BY
            agent_id
    ) AS config_scrapper ON config_scrapper.agent_id = agents.id
    LEFT JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            checks
        GROUP BY
            agent_id
    ) AS checks ON checks.agent_id = agents.id
    LEFT JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            playbook_runs
        GROUP BY
            agent_id
    ) AS playbook_runs ON playbook_runs.agent_id = agents.id
WHERE
  deleted_at IS NULL;

-- Revoke access tokens of an agent when it's deleted
CREATE OR REPLACE FUNCTION delete_access_tokens()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'DELETE' OR (TG_OP = 'UPDATE' AND OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL) THEN
    UPDATE access_tokens SET expires_at = NOW() WHERE person_id = (SELECT person_id FROM agents WHERE id = OLD.id);

    IF OLD.cleanup = TRUE OR NEW.cleanup = TRUE THEN
      PERFORM delete_agent_resources(OLD.id, NOW());
    END IF;
  END IF;

  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER delete_agent_access_tokens
AFTER UPDATE OR DELETE ON agents
FOR EACH ROW
EXECUTE PROCEDURE delete_access_tokens();

-- Mark agent resources as deleted
CREATE OR REPLACE FUNCTION delete_agent_resources(agentid UUID, delete_time TIMESTAMPTZ)
RETURNS VOID AS $$
BEGIN
    IF agentid IS NULL OR agentid = '00000000-0000-0000-0000-000000000000' THEN
        RETURN;
    END IF;

    UPDATE topologies SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE components SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE canaries SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE checks SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE config_scrapers SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE config_items SET deleted_at = delete_time WHERE agent_id = agentid;
    UPDATE logging_backends SET deleted_at = delete_time WHERE agent_id = agentid;

END
$$ LANGUAGE plpgsql;
