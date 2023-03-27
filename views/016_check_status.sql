CREATE OR REPLACE VIEW check_statuses_5m AS
SELECT
  date_trunc('minute', TIME) - (date_part('minute', TIME)::INT % 5) * INTERVAL '1 minute' AS interval_start,
  count(*) AS total_checks,
  count(*) FILTER (WHERE status = TRUE) AS successful_checks,
  count(*) FILTER (WHERE status = FALSE) AS failed_checks,
  SUM(duration) AS total_duration
FROM
  check_statuses
GROUP BY
  interval_start
ORDER BY
  interval_start DESC;