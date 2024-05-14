-- Add the hard relationship for records existing before the trigger was added
INSERT INTO config_relationships (config_id, related_id, relation)
SELECT parent.id "config_id", child.id "related_id", 'hard'
FROM config_items child
JOIN config_items parent 
  ON child.parent_id = parent.id
WHERE child.deleted_at IS NULL 
  AND parent.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM config_relationships cr
    WHERE cr.config_id = parent.id
      AND cr.related_id = child.id
      AND cr.relation = 'hard'
  );
