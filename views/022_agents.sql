CREATE OR REPLACE VIEW agents_summary AS
SELECT
    agents.*,
    configs.count as config_count,
    config_scrapper.count as config_scrapper_count,
    checks.count as checks_count,
    playbook_runs.count as playbook_runs_count
FROM
    agents
    INNER JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            config_items
        GROUP BY
            agent_id
    ) AS configs ON configs.agent_id = agents.id
    INNER JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            config_scrapers
        GROUP BY
            agent_id
    ) AS config_scrapper ON config_scrapper.agent_id = agents.id
    INNER JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            checks
        GROUP BY
            agent_id
    ) AS checks ON checks.agent_id = agents.id
    INNER JOIN (
        SELECT
            agent_id,
            COUNT(id) as count
        FROM
            playbook_runs
        GROUP BY
            agent_id
    ) AS playbook_runs ON playbook_runs.agent_id = agents.id