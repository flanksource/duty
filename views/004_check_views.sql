CREATE OR REPLACE VIEW
  check_names AS
SELECT
  id,
  canary_id,
type,
name,
status
FROM
  checks
where
  deleted_at is null
  AND silenced_at is null
ORDER BY
  name;

CREATE OR REPLACE VIEW
  check_labels AS
SELECT
  d.key,
  d.value
FROM
  checks
  JOIN json_each_text(labels::json) d on true
GROUP BY
  d.key,
  d.value
ORDER BY
  key,
  value;
