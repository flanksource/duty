UPDATE
  permissions
SET
  subject = COALESCE(team_id, person_id, notification_id),
  subject_type = CASE 
    WHEN team_id IS NOT NULL THEN 'team'
    WHEN person_id IS NOT NULL THEN 'person'
    WHEN notification_id IS NOT NULL THEN 'notification'
    ELSE 'group'
  END
WHERE
  subject IS NULL
  OR subject = '';

-- ALTER TABLE permissions ALTER COLUMN subject SET NOT NULL;
-- permission_group_summary
CREATE OR REPLACE VIEW permissions_group_summary AS
SELECT
  permissions.subject,
  permissions.subject_type,
  permissions.subject AS subject_label,
  permissions.action,
  permissions.object,
  permissions.deny
FROM
  permissions
WHERE
  subject_type = 'group';

-- permissions_summary (legacy view)
DROP VIEW IF EXISTS permissions_summary;

CREATE OR REPLACE VIEW permissions_summary AS
SELECT
    p.id,
    p.name,
    p.namespace,
    p.description,
    p.action,
    p.object,
    p.object_selector,
    p.deny,
    p.subject,
    p.subject_type,
    p.created_at,
    p.updated_at,
    p.deleted_at,
    p.created_by,
    p.updated_by,
    p.until,
    p.tags,
    p.agents,
    
    -- Person details (JSONB with id, name, email)
    CASE WHEN p.subject_type = 'person' THEN
        jsonb_build_object(
            'id', pe.id,
            'name', pe.name,
            'email', pe.email
        )
    ELSE NULL END AS person,
    
    -- Team details (id, namespace, name)
    CASE WHEN p.subject_type = 'team' THEN
        jsonb_build_object(
            'id', t.id,
            'namespace', t.source,  -- Using source as namespace
            'name', t.name
        )
    ELSE NULL END AS team,
    
    -- Canary details (id, namespace, name)
    CASE WHEN p.subject_type = 'canary' THEN
        jsonb_build_object(
            'id', c.id,
            'namespace', c.namespace,
            'name', c.name
        )
    ELSE NULL END AS canary,
    
    -- PermissionGroup details 
    CASE WHEN p.subject_type = 'group' THEN
        jsonb_build_object(
            'id', pg.id,
            'namespace', pg.namespace,
            'name', pg.name
        )
    ELSE NULL END AS group,
    
    -- Notification details (id, namespace, name)
    CASE WHEN p.subject_type = 'notification' THEN
        jsonb_build_object(
            'id', n.id,
            'namespace', n.namespace,
            'name', n.name
        )
    ELSE NULL END AS notification,
    
    -- Playbook details (id, namespace, name)
    CASE WHEN p.subject_type = 'playbook' THEN
        jsonb_build_object(
            'id', pb.id,
            'namespace', pb.namespace,
            'name', pb.name
        )
    ELSE NULL END AS playbook,
    
    -- Scraper details (id, namespace, name)
    CASE WHEN p.subject_type = 'scraper' THEN
        jsonb_build_object(
            'id', cs.id,
            'namespace', cs.namespace,
            'name', cs.name
        )
    ELSE NULL END AS scraper,
    
    -- Topology details (id, namespace, name)
    CASE WHEN p.subject_type = 'topology' THEN
        jsonb_build_object(
            'id', tp.id,
            'namespace', tp.namespace,
            'name', tp.name
        )
    ELSE NULL END AS topology
    
FROM permissions p
LEFT JOIN permission_groups pg ON p.subject_type = 'group' AND pg.name = p.subject AND pg.deleted_at IS NULL
LEFT JOIN people pe ON p.subject_type = 'person' AND pe.id::text = p.subject AND pe.deleted_at IS NULL
LEFT JOIN teams t ON p.subject_type = 'team' AND t.id::text = p.subject AND t.deleted_at IS NULL
LEFT JOIN canaries c ON p.subject_type = 'canary' AND c.id::text = p.subject AND c.deleted_at IS NULL
LEFT JOIN notifications n ON p.subject_type = 'notification' AND n.id::text = p.subject AND n.deleted_at IS NULL
LEFT JOIN playbooks pb ON p.subject_type = 'playbook' AND pb.id::text = p.subject AND pb.deleted_at IS NULL
LEFT JOIN config_scrapers cs ON p.subject_type = 'scraper' AND cs.id::text = p.subject AND cs.deleted_at IS NULL
LEFT JOIN topologies tp ON p.subject_type = 'topology' AND tp.id::text = p.subject AND tp.deleted_at IS NULL
WHERE p.deleted_at IS NULL;
