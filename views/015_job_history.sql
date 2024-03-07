-- We drop this first because of dependencies
DROP VIEW IF EXISTS integrations_with_status;

-- Intermediate view to get the latest job history status for each resource
CREATE OR REPLACE VIEW
  job_history_latest_status AS
WITH
  latest_job_history AS (
    SELECT
      job_history.resource_id,
      MAX(job_history.created_at) AS max_created_at
    FROM
      job_history
    GROUP BY
      job_history.resource_id
  )
SELECT
  job_history.*
FROM
  job_history
  JOIN latest_job_history ON job_history.resource_id = latest_job_history.resource_id
  AND job_history.created_at = latest_job_history.max_created_at;

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
  lr.time_start job_last_failed
FROM
  topologies
  LEFT JOIN job_history_latest_status ON topologies.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'topology'
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
    FROM checks
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
  lr.time_start job_last_failed
FROM
  canaries
  LEFT JOIN job_history_latest_status ON canaries.id::TEXT = job_history_latest_status.resource_id
  LEFT JOIN canaries_last_runtime ON canaries_last_runtime.canary_id = canaries.id
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
  lr.time_start job_last_failed
FROM
  config_scrapers
  LEFT JOIN job_history_latest_status ON config_scrapers.id::TEXT = job_history_latest_status.resource_id
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
SELECT
  notifications.id,
  notifications.title,
  notifications.events,
  notifications.filter,
  notifications.person_id,
  notifications.team_id,
  notifications.custom_services,
  notifications.created_at,
  notifications.updated_at,
  notifications.created_by,
  COUNT (event_queue.id) AS pending,
  ROUND(AVG(CASE WHEN notification_send_history.error IS NOT NULL THEN notification_send_history.duration_millis ELSE NULL END), 2) AS avg_duration_ms,
  COUNT (CASE WHEN notification_send_history.error IS NOT NULL THEN 1 END) AS failed,
  COUNT (CASE WHEN notification_send_history.error IS NULL THEN 1 END) AS sent,
  mode() WITHIN GROUP (ORDER BY notification_send_history.error) AS most_common_error
FROM
  notifications
  LEFT JOIN notification_send_history ON notifications.id = notification_send_history.notification_id
  LEFT JOIN event_queue ON 
    notifications.id::TEXT = event_queue.properties->>'notification_id' AND
    event_queue.name = 'notification.send'
WHERE
  notifications.deleted_at IS NULL
GROUP BY notifications.id;


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
  job_last_failed
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
  job_last_failed
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
  NULL
FROM
  logging_backends
)
SELECT combined.*, people.name AS creator_name, people.avatar AS creator_avatar, people.title AS creator_title, people.email AS creator_email FROM combined LEFT JOIN people ON combined.created_by = people.id;
