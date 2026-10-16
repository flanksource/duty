-- Only explicitly public UI properties belong here; backend boolean settings are configurations.
CREATE OR REPLACE FUNCTION property_type(name text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE
AS $$
  SELECT CASE WHEN name IN (
    'topology.disable',
    'health.disable',
    'incidents.disable',
    'config.disable',
    'logs.disable',
    'playbooks.disable',
    'applications.disable',
    'views.disable',
    'ai.disable',
    'agents.disable',
    'settings.connections.disable',
    'settings.users.disable',
    'settings.teams.disable',
    'settings.rules.disable',
    'settings.config_scraper.disable',
    'settings.topology.disable',
    'settings.health.disable',
    'settings.job_history.disable',
    'settings.feature_flags.disable',
    'settings.logging_backends.disable',
    'settings.event_queue_status.disable',
    'settings.organization_profile.disable',
    'settings.notifications.disable',
    'settings.playbooks.disable',
    'settings.integrations.disable',
    'settings.permissions.disable',
    'settings.artifacts.disable',
    'settings.mcp.disable',
    'settings.plugins.disable',
    'defaults.dashboard_view',
    'proxy.disable',
    'flanksource.ui.snippets',
    'ui.content_security_policy'
  ) THEN 'feature_flag' ELSE 'configuration' END;
$$;
