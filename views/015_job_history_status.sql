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

-- Template View
DROP VIEW IF EXISTS templates_with_status;

CREATE OR REPLACE VIEW
  templates_with_status AS
SELECT
  templates.*,
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
  templates
  LEFT JOIN job_history_latest_status ON templates.id::TEXT = job_history_latest_status.resource_id
  AND job_history_latest_status.resource_type = 'system_template';

-- Canaries View
DROP VIEW IF EXISTS canaries_with_status;

CREATE OR REPLACE VIEW
  canaries_with_status AS
SELECT
  canaries.*,
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
  AND job_history_latest_status.resource_type = 'canary';

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
  AND job_history_latest_status.resource_type = 'team';

-- Config scrapers View
DROP VIEW IF EXISTS config_scrapers_with_status;

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
  LEFT JOIN job_history_latest_status ON config_scrapers.id::TEXT = job_history_latest_status.resource_id;
