CREATE OR REPLACE VIEW checks_status_artifacts AS
SELECT
  check_statuses.check_id,
  check_statuses.status AS check_status,
  check_statuses.message AS check_message,
  check_statuses.error AS error,
  check_statuses.invalid AS invalid,
  check_statuses.time AS time,
  check_statuses.duration as duration,
  json_agg(
    json_build_object(
      'name', artifacts.filename,
      'size', artifacts.size,
      'id', artifacts.id
    )
  ) AS artifacts
FROM
  check_statuses
  LEFT JOIN artifacts ON check_statuses.check_id = artifacts.check_id
  AND check_statuses.time = artifacts.check_time
GROUP BY
  check_statuses.time,
  check_statuses.check_id
