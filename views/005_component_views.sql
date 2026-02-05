-- components by check
DROP FUNCTION IF EXISTS lookup_components_by_check;
CREATE OR REPLACE FUNCTION lookup_components_by_check (id uuid) RETURNS setof components AS
$$
SELECT * FROM components WHERE id IN (
  SELECT component_id FROM check_component_relationships
  WHERE check_id = $1
)
$$ LANGUAGE sql;

-- lookup_component_by_property
CREATE OR REPLACE function lookup_component_by_property (text, text) returns setof components as $$
begin
  return query
    select * from components where deleted_at is null AND properties @>  jsonb_build_array(json_build_object('name', $1, 'text', $2));
end;
$$ language plpgsql;

-- lookup_components_by_config
DROP FUNCTION IF EXISTS lookup_components_by_config;

CREATE OR REPLACE function lookup_components_by_config (id text) returns table (
  component_id UUID,
  name TEXT,
  type TEXT,
  icon TEXT,
  role TEXT
) as $$
begin
  RETURN QUERY
	  SELECT components.id as component_id , components.name, components.type, components.icon, 'left' as role
	  FROM config_component_relationships
	  INNER JOIN  components on components.id = config_component_relationships.component_id
	  WHERE config_component_relationships.config_id = $1::uuid;
end;
$$ language plpgsql;


DROP VIEW IF EXISTS component_names;
CREATE OR REPLACE VIEW component_names AS
  SELECT
    id,
    path,
    external_id,
    type,
    name,
    created_at,
    updated_at,
    icon,
    parent_id
  FROM
    components
  WHERE
    deleted_at is null
    AND hidden != true
  ORDER BY
    name,
    external_id;

DROP VIEW IF EXISTS component_names_all;
CREATE OR REPLACE VIEW component_names_all AS
SELECT
  id,
  path,
  external_id,
  type,
  name,
  created_at,
  updated_at,
  deleted_at,
  icon,
  parent_id
FROM
  components
WHERE
  hidden != true
ORDER BY
  name,
  external_id;

CREATE OR REPLACE VIEW component_labels AS
  SELECT
    d.key,
    d.value
  FROM
    components
    JOIN json_each_text(labels::json) d on true
  GROUP BY
    d.key,
    d.value
  ORDER BY
    key,
    value;

-- component_labels_keys
DROP VIEW IF EXISTS component_labels_keys;
CREATE OR REPLACE VIEW component_labels_keys AS
  SELECT DISTINCT 'label:' || jsonb_object_keys(labels) AS "key" FROM components;

DROP VIEW IF EXISTS components_with_logs;
CREATE OR REPLACE VIEW components_with_logs AS
  SELECT
    id,
    name,
    agent_id,
    icon,
    type
  FROM
    components
  WHERE
    deleted_at IS NULL
    AND log_selectors IS NOT NULL;

-- TODO stop the recursion once max_depth is reached.level <= max_depth;
DROP FUNCTION if exists lookup_component_children;

CREATE OR REPLACE FUNCTION lookup_component_children (id text, max_depth int)
RETURNS TABLE (child_id UUID, parent_id UUID, level int) AS $$
BEGIN
    IF max_depth < 0 THEN
        max_depth = 10;
    END IF;
    RETURN QUERY
        WITH RECURSIVE children AS (
            SELECT components.id as child_id, components.parent_id, 0 as level
            FROM components
            WHERE components.id = $1::uuid
            UNION ALL
            SELECT m.id as child_id, m.parent_id, c.level + 1 as level
            FROM components m
            JOIN children c ON m.parent_id = c.child_id
        )
        SELECT children.child_id, children.parent_id, children.level FROM children
        WHERE children.level <= max_depth;
END;
$$ language plpgsql;

DROP FUNCTION IF EXISTS lookup_component_relations;
CREATE OR REPLACE FUNCTION lookup_component_relations (component_id text)
RETURNS TABLE (id UUID) AS $$
BEGIN
    RETURN QUERY
        SELECT cr.relationship_id AS id FROM component_relationships cr WHERE cr.component_id = $1::UUID
        UNION
        SELECT cr.component_id as id FROM component_relationships cr WHERE cr.relationship_id = $1::UUID;
END;
$$ language plpgsql;

CREATE OR REPLACE VIEW component_types AS
  SELECT distinct on (type) type
  FROM components ORDER BY type asc;

DROP FUNCTION IF EXISTS lookup_component_config_id_related_components;
CREATE OR REPLACE FUNCTION lookup_component_config_id_related_components (
  component_id TEXT
)
RETURNS TABLE (id UUID) AS $$
BEGIN
    RETURN QUERY
        WITH config_id_paths AS (
            SELECT config_items.id
            FROM config_items
            WHERE starts_with(path, (SELECT CONCAT(config_items.path, '.', config_items.id) FROM config_items WHERE config_items.id IN (SELECT config_id FROM components WHERE components.id = $1::UUID)))
        )
        SELECT components.id 
        FROM components 
        INNER JOIN config_id_paths ON config_id_paths.id = components.config_id;
END;
$$ LANGUAGE plpgsql
