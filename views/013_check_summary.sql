-- Materialized view for check status summary
CREATE MATERIALIZED VIEW IF NOT EXISTS
  check_status_summary AS
SELECT
  check_id,
  PERCENTILE_DISC(0.99) WITHIN GROUP (
    ORDER BY
      duration
  ) as p99,
  COUNT(*) FILTER (
    WHERE
      status = TRUE
  ) as passed,
  COUNT(*) FILTER (
    WHERE
      status = FALSE
  ) as failed,
  MAX(time) as last_check,
  MAX(time) FILTER (
    WHERE
      status = TRUE
  ) as last_pass,
  MAX(time) FILTER (
    WHERE
      status = FALSE
  ) as last_fail
FROM
  check_statuses
WHERE
  time > (NOW() at TIME ZONE 'utc' - Interval '1 hour')
GROUP BY
  check_id;

-- For last transition
CREATE
OR REPLACE FUNCTION update_last_transition_time_for_check () RETURNS TRIGGER AS $$
BEGIN
    NEW.last_transition_time = NOW();
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE
OR REPLACE TRIGGER checks_last_transition_time BEFORE
UPDATE ON checks FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE PROCEDURE update_last_transition_time_for_check ();

-- check summary view
CREATE OR REPLACE VIEW check_summary AS
  WITH check_component_relationship_by_check AS (
  SELECT
    check_id,
    json_agg(component_id) AS components
  FROM
    check_component_relationships
  GROUP BY
    check_id
  )
  SELECT
    checks.id,
    checks.canary_id,
    json_build_object(
      'passed', check_status_summary.passed,
      'failed', check_status_summary.failed,
      'last_pass', check_status_summary.last_pass,
      'last_fail', check_status_summary.last_fail
    ) AS uptime,
    json_build_object('p99', check_status_summary.p99) AS latency,
    checks.last_transition_time,
    checks.type,
    checks.icon,
    checks.name,
    checks.status,
    checks.description,
    canaries.namespace,
    canaries.name as canary_name,
    canaries.labels,
    checks.severity,
    checks.owner,
    checks.last_runtime,
    checks.created_at,
    checks.updated_at,
    checks.deleted_at,
    checks.silenced_at,
    check_component_relationship_by_check.components
  FROM
    checks
    LEFT JOIN check_component_relationship_by_check ON checks.id = check_component_relationship_by_check.check_id
    INNER JOIN canaries ON checks.canary_id = canaries.id
    INNER JOIN check_status_summary ON checks.id = check_status_summary.check_id;

-- Check summary by component
DROP FUNCTION IF EXISTS check_summary_for_component;

CREATE OR REPLACE FUNCTION check_summary_for_component(id uuid) RETURNS setof check_summary
AS $$
  BEGIN
    RETURN QUERY
    SELECT check_summary.* FROM check_summary
    INNER JOIN check_component_relationships
      ON check_component_relationships.check_id = check_summary.id 
    WHERE check_component_relationships.component_id = $1;
  END;
$$ language plpgsql;