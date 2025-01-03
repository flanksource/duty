UPDATE
  permissions
SET
  subject = COALESCE(team_id, person_id, notification_id)
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

-- permission_summary
CREATE OR REPLACE VIEW permissions_summary AS
SELECT
  permissions.subject,
  permissions.subject_type,
  COALESCE(people.name, teams.name, notifications.name) AS subject_label,
  permissions.action,
  permissions.object,
  permissions.deny
FROM
  permissions
  LEFT JOIN people ON permissions.subject_type = 'person'
    AND people.id = permissions.subject::uuid
  LEFT JOIN teams ON permissions.subject_type = 'team'
    AND teams.id = permissions.subject::uuid
  LEFT JOIN notifications ON permissions.subject_type = 'notification'
    AND notifications.id = permissions.subject::uuid
WHERE
  subject_type != 'group'
UNION
SELECT
  *
FROM
  permissions_group_summary;

