WITH duplicates AS (
    SELECT
        name,
        namespace,
        category,
        ROW_NUMBER() OVER (PARTITION BY name, namespace, category ORDER BY id) as row_num
    FROM playbooks
)
UPDATE playbooks
SET name = playbooks.name || '_' || right(id::text, 4)
FROM duplicates
WHERE playbooks.name = duplicates.name
  AND playbooks.namespace = duplicates.namespace
  AND playbooks.category = duplicates.category
  AND duplicates.row_num > 1;
