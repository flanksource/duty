CREATE OR REPLACE VIEW check_statuses_5m AS
SELECT
  check_statuses.check_id,
  date_trunc('minute', "time") - (date_part('minute', "time")::INT % 5) * INTERVAL '1 minute' AS created_at,
  count(*) AS total_checks,
  count(*) FILTER (WHERE check_statuses.status = TRUE) AS passed,
  count(*) FILTER (WHERE check_statuses.status = FALSE) AS failed,
  SUM(duration) AS total_duration
FROM
  check_statuses LEFT JOIN checks ON check_statuses.check_id = checks.id
WHERE
  checks.created_at > now() - INTERVAL '1 day'
GROUP BY
  check_id, date_trunc('minute', "time") - (date_part('minute', "time") :: INT % 5) * INTERVAL '1 minute'
ORDER BY
  check_id, created_at DESC;