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

-- checks_labels_keys
DROP VIEW IF EXISTS checks_labels_keys;
CREATE OR REPLACE VIEW checks_labels_keys AS
  SELECT DISTINCT 'label:' || jsonb_object_keys(labels) AS "key" FROM checks;