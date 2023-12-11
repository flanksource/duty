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
DROP VIEW IF EXISTS topologies_with_status CASCADE;

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
  job_history_latest_status.created_at job_created_at
FROM
  topologies
  LEFT JOIN job_history_latest_status ON topologies.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'topology'
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
  job_history_latest_status.created_at job_created_at
FROM
  canaries
  LEFT JOIN job_history_latest_status ON canaries.id::TEXT = job_history_latest_status.resource_id
  LEFT JOIN canaries_last_runtime ON canaries_last_runtime.canary_id = canaries.id
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
  job_history_latest_status.created_at job_created_at
FROM
  teams
  LEFT JOIN job_history_latest_status ON teams.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'team'
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
  job_history_latest_status.created_at job_created_at
FROM
  config_scrapers
  LEFT JOIN job_history_latest_status ON config_scrapers.id::TEXT = job_history_latest_status.resource_id
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
  job_history_latest_status.status job_status,
  job_history_latest_status.details job_details,
  job_history_latest_status.duration_millis job_duration_millis,
  job_history_latest_status.time_start job_time_start
FROM
  notifications
  LEFT JOIN job_history_latest_status ON notifications.id::TEXT = job_history_latest_status.resource_id
WHERE
  notifications.deleted_at IS NULL;
