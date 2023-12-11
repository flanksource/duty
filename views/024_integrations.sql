CREATE OR REPLACE VIEW integrations_list AS
SELECT id, name, description, 'scrapers' AS integration_type, source, agent_id, created_at, updated_at, deleted_at, created_by FROM config_scrapers_with_status UNION
SELECT id, name, '', 'topologies' AS integration_type, source, agent_id, created_at, updated_at, deleted_at, created_by FROM topologies_with_status UNION
SELECT id, name, '', 'logging_backends' AS integration_type,  source, agent_id, created_at, updated_at, deleted_at, created_by FROM logging_backends;
