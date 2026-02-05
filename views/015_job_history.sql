-- We drop this first because of dependencies
DROP VIEW IF EXISTS integrations_with_status;

-- Intermediate view to get the latest job history status for each resource
CREATE OR REPLACE VIEW
  job_history_latest_status AS

WITH
  latest_job_history AS (
    SELECT
      jh.*,
      ROW_NUMBER() OVER (
        PARTITION BY jh.resource_id, jh.resource_type, jh.name
        ORDER BY jh.created_at DESC, jh.id DESC
      ) AS rn
    FROM
      job_history jh
  )
SELECT
  latest_job_history.*
FROM
  latest_job_history
WHERE
  rn = 1;

-- Topologies with job status
DROP VIEW IF EXISTS topologies_with_status;

CREATE OR REPLACE VIEW
  topologies_with_status AS
SELECT
  topologies.*,
  job_history_latest_status.name job_name,
  job_history_latest_status.success_count job_success_count,
  job_history_latest_status.error_count job_error_count,
  job_history_latest_status.details job_details,
  job_history_latest_status.hostname job_hostname,
  job_history_latest_status.duration_millis job_duration_millis,
  job_history_latest_status.resource_type job_resource_type,
  job_history_latest_status.status job_status,
  job_history_latest_status.time_start job_time_start,
  job_history_latest_status.time_end job_time_end,
  job_history_latest_status.created_at job_created_at,
  lr.time_start job_last_failed,
  json_build_object(
      'id', agents.id,
      'name', agents.name
    ) as agent
FROM
  topologies
  LEFT JOIN job_history_latest_status ON topologies.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'topology'
  LEFT JOIN agents ON agents.id = topologies.agent_id
  LEFT JOIN (
    SELECT
      resource_id,
      MAX(time_start) as time_start
    FROM job_history as js_last_failed
    WHERE js_last_failed.status = 'FAILED' AND js_last_failed.resource_type = 'topology'
    GROUP BY js_last_failed.resource_id
  ) lr ON lr.resource_id = topologies.id::TEXT
WHERE
  topologies.deleted_at IS NULL;

-- Canaries View
DROP VIEW IF EXISTS canaries_with_status;
CREATE OR REPLACE VIEW canaries_with_status AS
WITH canaries_last_runtime AS (
    SELECT MAX(last_runtime) as last_runtime, canary_id
    FROM checks_unlogged
    GROUP BY canary_id
)
SELECT
  canaries.id,
  canaries.name,
  canaries.namespace,
  canaries.spec->>'interval' AS interval,
  canaries.spec->>'schedule' AS schedule,
  canaries.labels,
  canaries.source,
  canaries.created_by,
  canaries.created_at,
  canaries.deleted_at,
  canaries_last_runtime.last_runtime,
  job_history_latest_status.name job_name,
  job_history_latest_status.success_count job_success_count,
  job_history_latest_status.error_count job_error_count,
  job_history_latest_status.details job_details,
  job_history_latest_status.hostname job_hostname,
  job_history_latest_status.duration_millis job_duration_millis,
  job_history_latest_status.resource_type job_resource_type,
  job_history_latest_status.status job_status,
  job_history_latest_status.time_start job_time_start,
  job_history_latest_status.time_end job_time_end,
  job_history_latest_status.created_at job_created_at,
  lr.time_start job_last_failed,
  json_build_object(
      'id', agents.id,
      'name', agents.name
    ) as agent
FROM
  canaries
  LEFT JOIN job_history_latest_status ON canaries.id::TEXT = job_history_latest_status.resource_id
  LEFT JOIN canaries_last_runtime ON canaries_last_runtime.canary_id = canaries.id
  LEFT JOIN agents ON agents.id = canaries.agent_id
  LEFT JOIN (
    SELECT
      resource_id,
      MAX(time_start) as time_start
    FROM job_history as js_last_failed
    WHERE js_last_failed.status = 'FAILED' AND js_last_failed.resource_type = 'canary'
    GROUP BY js_last_failed.resource_id
  ) lr ON lr.resource_id = canaries.id::TEXT
WHERE
  canaries.deleted_at IS NULL;

-- Teams View
DROP VIEW IF EXISTS teams_with_status;

CREATE OR REPLACE VIEW
  teams_with_status AS
SELECT
  teams.*,
  job_history_latest_status.name job_name,
  job_history_latest_status.success_count job_success_count,
  job_history_latest_status.error_count job_error_count,
  job_history_latest_status.details job_details,
  job_history_latest_status.hostname job_hostname,
  job_history_latest_status.duration_millis job_duration_millis,
  job_history_latest_status.resource_type job_resource_type,
  job_history_latest_status.status job_status,
  job_history_latest_status.time_start job_time_start,
  job_history_latest_status.time_end job_time_end,
  job_history_latest_status.created_at job_created_at,
  lr.time_start job_last_failed
FROM
  teams
  LEFT JOIN job_history_latest_status ON teams.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'team'
  LEFT JOIN (
    SELECT
      resource_id,
      MAX(time_start) as time_start
    FROM job_history as js_last_failed
    WHERE js_last_failed.status = 'FAILED' AND js_last_failed.resource_type = 'team'
    GROUP BY js_last_failed.resource_id
  ) lr ON lr.resource_id = teams.id::TEXT
WHERE
  teams.deleted_at IS NULL;

-- Config scrapers View
DROP VIEW IF EXISTS config_scrapers_with_status CASCADE;

CREATE OR REPLACE VIEW
  config_scrapers_with_status AS
SELECT
  config_scrapers.*,
  job_history_latest_status.name job_name,
  job_history_latest_status.success_count job_success_count,
  job_history_latest_status.error_count job_error_count,
  job_history_latest_status.details job_details,
  job_history_latest_status.hostname job_hostname,
  job_history_latest_status.duration_millis job_duration_millis,
  job_history_latest_status.resource_type job_resource_type,
  job_history_latest_status.status job_status,
  job_history_latest_status.time_start job_time_start,
  job_history_latest_status.time_end job_time_end,
  job_history_latest_status.created_at job_created_at,
  lr.time_start job_last_failed,
  json_build_object(
      'id', agents.id,
      'name', agents.name
    ) as agent
FROM
  config_scrapers
  LEFT JOIN LATERAL (
    SELECT *
    FROM job_history_latest_status jh
    WHERE jh.resource_id = config_scrapers.id::TEXT
    AND jh.resource_type = 'config_scraper'
    ORDER BY jh.created_at DESC
    LIMIT 1
  ) job_history_latest_status ON TRUE
  LEFT JOIN agents ON agents.id = config_scrapers.agent_id
  LEFT JOIN (
    SELECT
      resource_id,
      MAX(time_start) as time_start
    FROM job_history as js_last_failed
    WHERE js_last_failed.status = 'FAILED' AND js_last_failed.resource_type = 'config_scraper'
    GROUP BY js_last_failed.resource_id
  ) lr ON lr.resource_id = config_scrapers.id::TEXT
WHERE
  config_scrapers.deleted_at IS NULL;

DROP VIEW IF EXISTS job_history_names;
CREATE OR REPLACE VIEW job_history_names AS
  SELECT distinct on (name) name
  FROM job_history;

-- Notifications with job history
DROP VIEW IF EXISTS notifications_summary;

CREATE OR REPLACE VIEW notifications_summary AS
WITH notification_send_summary AS (
  SELECT
    notification_id,
    ROUND(AVG(CASE WHEN error IS NOT NULL THEN duration_millis ELSE NULL END), 2) AS avg_duration_ms,
    COUNT(CASE WHEN error IS NOT NULL THEN 1 END) AS failed,
    COUNT(CASE WHEN status = 'sent' THEN 1 END) AS sent,
    mode() WITHIN GROUP (ORDER BY error) AS most_common_error,
    MAX(CASE WHEN error IS NOT NULL THEN created_at ELSE NULL END) AS last_failed_at
  FROM
    notification_send_history
  WHERE
    source_event <> 'notification.watchdog'
  GROUP BY notification_id
)
SELECT
  notifications.id,
  notifications.name,
  COALESCE(notifications.namespace, '') AS namespace,
  notifications.error,
  notifications.error_at,
  notifications.title,
  notifications.events,
  notifications.filter,
  notifications.person_id,
  notifications.team_id,
  notifications.custom_services,
  notifications.created_at,
  notifications.updated_at,
  notifications.created_by,
  notifications.source,
  notifications.repeat_interval,
  notifications.wait_for,
  COUNT (event_queue.id) AS pending,
  notification_send_summary.avg_duration_ms,
  COALESCE(notification_send_summary.failed, 0) AS failed,
  COALESCE(notification_send_summary.sent, 0) AS sent,
  notification_send_summary.most_common_error,
  notification_send_summary.last_failed_at
FROM
  notifications
  LEFT JOIN notification_send_summary ON notifications.id = notification_send_summary.notification_id
  LEFT JOIN event_queue ON
    notifications.id::TEXT = event_queue.properties->>'notification_id' AND
    event_queue.name = 'notification.send' AND
    event_queue.attempts < 4
WHERE
  notifications.deleted_at IS NULL
GROUP BY notifications.id,
notification_send_summary.avg_duration_ms,
notification_send_summary.failed,
notification_send_summary.sent,
notification_send_summary.most_common_error,
notification_send_summary.last_failed_at;

CREATE VIEW integrations_with_status AS
WITH combined AS (
SELECT
  id,
  NAME,
  description,
  'scrapers' AS integration_type,
  source,
  agent_id,
  created_at,
  updated_at,
  deleted_at,
  created_by,
  job_name,
  job_success_count,
  job_error_count,
  job_details,
  job_hostname,
  job_duration_millis,
  job_resource_type,
  job_status,
  job_time_start,
  job_time_end,
  job_created_at,
  job_last_failed,
  agent::jsonb
FROM
  config_scrapers_with_status
UNION
SELECT
  id,
  NAME,
  '',
  'topologies' AS integration_type,
  source,
  agent_id,
  created_at,
  updated_at,
  deleted_at,
  created_by,
  job_name,
  job_success_count,
  job_error_count,
  job_details,
  job_hostname,
  job_duration_millis,
  job_resource_type,
  job_status,
  job_time_start,
  job_time_end,
  job_created_at,
  job_last_failed,
  agent::jsonb
FROM
  topologies_with_status
UNION
SELECT
  id,
  NAME,
  '',
  'logging_backends' AS integration_type,
  source,
  agent_id,
  created_at,
  updated_at,
  deleted_at,
  created_by,
  '',
  0,
  0,
  NULL,
  '',
  0,
  NULL,
  '',
  NULL,
  NULL,
  NULL,
  NULL,
  NULL
FROM
  logging_backends
)
SELECT combined.*, people.name AS creator_name, people.avatar AS creator_avatar, people.title AS creator_title, people.email AS creator_email FROM combined LEFT JOIN people ON combined.created_by = people.id;

DROP VIEW IF EXISTS job_histories;

CREATE OR REPLACE VIEW job_histories
AS SELECT
  job_history.*,
  COALESCE(
    components.name,
    config_scrapers.name,
    topologies.name,
    canaries.name,
    job_history.resource_id
  ) as resource_name,
  json_build_object(
    'id', agents.id,
    'name', agents.name
  ) as agent
FROM job_history
LEFT JOIN components ON job_history.resource_id = components.id::TEXT AND job_history.resource_type = 'components'
LEFT JOIN config_scrapers ON job_history.resource_id = config_scrapers.id::TEXT AND job_history.resource_type = 'config_scraper'
LEFT JOIN canaries ON job_history.resource_id = canaries.id::TEXT AND job_history.resource_type = 'canary'
LEFT JOIN topologies ON job_history.resource_id = topologies.id::TEXT AND job_history.resource_type = 'topology'
LEFT JOIN agents ON job_history.agent_id = agents.id;
