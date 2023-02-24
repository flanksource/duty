DROP VIEW IF EXISTS configs CASCADE;

CREATE or REPLACE VIEW configs AS
  SELECT
    ci.id,
    ci.scraper_id,
    ci.config_type,
    ci.external_id,
    ci.external_type,
    ci.name,
    ci.namespace,
    ci.description,
    ci.source,
    ci.tags,
    ci.created_by,
    ci.created_at,
    ci.updated_at,
    ci.deleted_at,
    ci.cost_per_minute,
    ci.cost_total_1d,
    ci.cost_total_7d,
    ci.cost_total_30d,
    analysis,
    changes
  FROM config_items as ci
    full join (
      SELECT config_id,
        json_agg(json_build_object('analyzer',analyzer,'analysis_type',analysis_type,'severity',severity)) as analysis
      FROM config_analysis
      GROUP BY  config_id
    ) as ca on ca.config_id = ci.id
    full join (
      SELECT config_id,
        json_agg(total) as changes
      FROM
      (SELECT config_id,json_build_object('change_type',change_type, 'severity', severity, 'total', count(*)) as total FROM config_changes GROUP BY config_id, change_type, severity) as config_change_types
      GROUP BY  config_id
    ) as cc on cc.config_id = ci.id;


CREATE or REPLACE VIEW config_names AS
  SELECT id, config_type, external_id, name FROM config_items;

CREATE or REPLACE VIEW config_types AS
  SELECT DISTINCT config_type FROM config_items;

CREATE or REPLACE VIEW analyzer_types AS
  SELECT DISTINCT analyzer FROM config_analysis;

CREATE or REPLACE VIEW analysis_types AS
  SELECT DISTINCT analysis_type FROM config_analysis;

CREATE or REPLACE VIEW change_types AS
  SELECT DISTINCT change_type FROM config_changes;



-- lookup_config_children
-- TODO stop the recursion once max_depth is reached.level <= max_depth;
CREATE OR REPLACE FUNCTION lookup_config_children(id text, max_depth int)
RETURNS TABLE(
    child_id UUID,
    parent_id UUID,
    level int
) AS $$
BEGIN
    IF max_depth < 0 THEN
        max_depth = 10;
    END IF;
    RETURN QUERY
        WITH RECURSIVE children AS (
            SELECT config_items.id as child_id, config_items.parent_id, 0 as level
            FROM config_items
            WHERE config_items.id = $1::uuid
            UNION ALL
            SELECT m.id as child_id, m.parent_id, c.level + 1 as level
            FROM config_items m
            JOIN children c ON m.parent_id = c.child_id
        )
        SELECT children.child_id, children.parent_id, children.level FROM children
        WHERE children.level <= max_depth;
END;
$$
language plpgsql;

-- lookup_config_relations
CREATE OR REPLACE FUNCTION lookup_config_relations(config_id text)
RETURNS TABLE (
    id UUID
) AS $$
BEGIN
    RETURN QUERY
        SELECT cr.related_id AS id FROM config_relationships cr WHERE cr.config_id = $1::UUID
        UNION
        SELECT cr.config_id as id FROM config_relationships cr WHERE cr.related_id = $1::UUID;
END;
$$
language plpgsql;


-- lookup_configs_by_component
CREATE OR REPLACE function lookup_configs_by_component(id text)
returns table (
  config_id UUID,
  name TEXT,
  type TEXT,
  icon TEXT,
  role TEXT
)
as
$$
begin
  RETURN QUERY
	  SELECT config_items.id as config_id, config_items.name, config_items.config_type, config_items.icon, 'left' as role
	  FROM config_component_relationships
	  INNER JOIN  config_items on config_items.id = config_component_relationships.config_id
	  WHERE config_component_relationships.component_id = $1::uuid;
end;
$$
language plpgsql;

-- lookup_changes_by_component
CREATE OR REPLACE function lookup_changes_by_component(id text)
RETURNS SETOF config_changes as
$$
begin
  RETURN QUERY select * from config_changes where config_id in (select config_id from lookup_configs_by_component($1));
end;
$$
language plpgsql;


-- lookup_related_configs
DROP FUNCTION IF EXISTS lookup_related_configs;
CREATE OR REPLACE function lookup_related_configs(id text)
returns table (
  config_id UUID,
  name TEXT,
  type TEXT,
  icon TEXT,
  role TEXT,
  relation TEXT
)
as
$$
begin

  RETURN QUERY
	  SELECT parent.id as config_id, parent.name, parent.config_type, parent.icon, 'parent' as role, null
	  FROM config_items
	  INNER JOIN  config_items parent on config_items.parent_id = parent.id
	  WHERE config_items.id = $1::uuid
	UNION
		  SELECT config_items.id as config_id, config_items.name, config_items.config_type, config_items.icon, 'left' as role, config_relationships.relation
		  FROM config_relationships
		  INNER JOIN  config_items on config_items.id = config_relationships.related_id
		  WHERE config_relationships.config_id = $1::uuid
	UNION
		  SELECT config_items.id as config_id, config_items.name, config_items.config_type, config_items.icon, 'right' as role , config_relationships.relation
		  FROM config_relationships
		  INNER JOIN  config_items on config_items.id = config_relationships.config_id
		  WHERE config_relationships.related_id = $1::uuid;
end;
$$
language plpgsql;

-- changes_by_component
DROP VIEW IF EXISTS changes_by_component;
CREATE OR REPLACE VIEW changes_by_component AS
	SELECT config_changes.config_id, configs.name, configs.config_type, configs.external_type, change_type,
         config_changes.created_at,config_changes.created_by, config_changes.id as change_id, config_changes.severity, component_id
  FROM config_changes
  INNER JOIN config_component_relationships relations on relations.config_id = config_changes.config_id
  INNER JOIN config_items  configs on configs.id = config_changes.config_id;

-- config_tags
DROP VIEW IF EXISTS config_tags;
CREATE OR REPLACE VIEW config_tags AS
  SELECT d.key, d.value
  FROM configs JOIN json_each_text(tags::json) d ON true GROUP BY d.key, d.value ORDER BY key, value;