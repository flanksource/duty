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

CREATE OR REPLACE VIEW
  checks_by_component AS
SELECT
  check_component_relationships.component_id,
  checks.id,
  checks.type,
  checks.name,
  checks.severity,
  checks.status
from
  check_component_relationships
  INNER JOIN checks ON checks.id = check_component_relationships.check_id
WHERE
  check_component_relationships.deleted_at is null;

-- check_summary_by_component
CREATE OR REPLACE VIEW
  check_summary_by_component AS
WITH cte as (
    SELECT
        component_id, status, COUNT(*) AS count
    FROM
      checks_by_component
    GROUP BY
      component_id, status
)
SELECT component_id, json_object_agg(status, count) AS checks
FROM cte GROUP BY component_id;
