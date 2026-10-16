-- Rename existing UI settings once; new public properties need no registration.
-- An explicitly configured public key wins if both names already exist.
UPDATE properties AS old
SET name = 'public.' || old.name
WHERE old.name IN (
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
)
AND NOT EXISTS (
    SELECT 1 FROM properties AS current WHERE current.name = 'public.' || old.name
);

CREATE OR REPLACE VIEW feature_flags AS
SELECT name, value, created_by, created_at, updated_at, deleted_at
FROM properties
WHERE name LIKE 'public.%' AND deleted_at IS NULL;
