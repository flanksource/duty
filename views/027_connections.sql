DROP VIEW IF EXISTS connections_list;
CREATE OR REPLACE VIEW connections_list AS
  SELECT
    id,
    name,
    namespace,
    type,
    CASE
      WHEN (string_to_array(url, '://'))[1] IN ('bark', 'discord', 'smtp', 'gotify', 'googlechat', 'ifttt', 'join', 'mattermost', 'matrix', 'ntfy', 'opsgenie', 'pushbullet', 'pushover', 'rocketchat', 'slack', 'teams', 'telegram', 'zulip') THEN 'notification'
      ELSE ''
    END AS category,
    created_by,
    created_at,
    updated_at
  FROM
    connections
  WHERE
    deleted_at IS NULL
  ORDER BY
    created_at;
