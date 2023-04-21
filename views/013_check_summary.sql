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
