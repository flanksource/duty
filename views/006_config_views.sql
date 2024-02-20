DROP VIEW IF EXISTS configs CASCADE;

CREATE or REPLACE VIEW configs AS
  SELECT
    ci.id,
    ci.scraper_id,
    ci.config_class,
    ci.external_id,
    ci.type,
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
    ci.agent_id,
    analysis,
    changes
  FROM config_items as ci
    full join (
      SELECT config_id,
        json_agg(json_build_object('analyzer',analyzer,'analysis_type',analysis_type,'severity',severity)) as analysis
      FROM config_analysis
      WHERE config_analysis.status = 'open'
      GROUP BY  config_id
    ) as ca on ca.config_id = ci.id
    full join (
      SELECT config_id,
        json_agg(total) as changes
      FROM
      (SELECT config_id,json_build_object('change_type',change_type, 'severity', severity, 'total', count(*)) as total FROM config_changes GROUP BY config_id, change_type, severity) as config_change_types
      GROUP BY  config_id
    ) as cc on cc.config_id = ci.id;


DROP VIEW IF EXISTS config_names;
CREATE or REPLACE VIEW config_names AS
  SELECT id, type, external_id, name FROM config_items ORDER BY name;

DROP VIEW IF EXISTS config_types;
CREATE or REPLACE VIEW config_types AS
  SELECT DISTINCT type FROM config_items ORDER BY type;

CREATE or REPLACE VIEW config_classes AS
  SELECT DISTINCT config_class FROM config_items ORDER BY config_class;

CREATE or REPLACE VIEW analyzer_types AS
  SELECT DISTINCT analyzer FROM config_analysis ORDER BY analyzer;

CREATE or REPLACE VIEW analysis_types AS
  SELECT DISTINCT analysis_type FROM config_analysis ORDER BY analysis_type ;

CREATE or REPLACE VIEW change_types AS
  SELECT DISTINCT change_type FROM config_changes ORDER BY change_type;

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
DROP FUNCTION IF EXISTS lookup_configs_by_component;
CREATE OR REPLACE FUNCTION lookup_configs_by_component(id text)
returns table (
  config_id UUID,
  name TEXT,
  type TEXT,
  icon TEXT,
  role TEXT,
  deleted_at TIMESTAMP WITH TIME ZONE
)
as
$$
begin
  RETURN QUERY
	  SELECT config_items.id as config_id, config_items.name, config_items.config_class, config_items.icon, 'left' as role, config_items.deleted_at
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

-- lookup analysis by component
CREATE OR REPLACE FUNCTION lookup_analysis_by_component(id text)
RETURNS SETOF config_analysis AS
$$
BEGIN
  RETURN QUERY
    SELECT * FROM config_analysis 
    WHERE config_id IN (
      SELECT config_id FROM lookup_configs_by_component($1)
    );
END;
$$
LANGUAGE plpgsql;

-- lookup_related_configs
DROP FUNCTION IF EXISTS lookup_related_configs;
CREATE OR REPLACE function lookup_related_configs(id text)
returns table (
  config_id UUID,
  name TEXT,
  type TEXT,
  icon TEXT,
  role TEXT,
  relation TEXT,
  deleted_at TIMESTAMP
)
as
$$
begin

  RETURN QUERY
	  SELECT parent.id as config_id, parent.name, parent.config_class, parent.icon, 'parent' as role, null, parent.deleted_at
	  FROM config_items
	  INNER JOIN  config_items parent on config_items.parent_id = parent.id
	UNION
		  SELECT config_items.id as config_id, config_items.name, config_items.config_class, config_items.icon, 'left' as role, config_relationships.relation, config_items.deleted_at
		  FROM config_relationships
		  INNER JOIN  config_items on config_items.id = config_relationships.related_id
		  WHERE config_relationships.config_id = $1::uuid
	UNION
		  SELECT config_items.id as config_id, config_items.name, config_items.config_class, config_items.icon, 'right' as role , config_relationships.relation, config_items.deleted_at
		  FROM config_relationships
		  INNER JOIN  config_items on config_items.id = config_relationships.config_id
		  WHERE config_relationships.related_id = $1::uuid;
end;
$$
language plpgsql;

-- changes_by_component
DROP VIEW IF EXISTS changes_by_component;
CREATE OR REPLACE VIEW changes_by_component AS
SELECT config_changes.id, config_changes.config_id, configs.name, configs.config_class, configs.type, change_type,
     config_changes.created_at,config_changes.created_by, config_changes.id as change_id, config_changes.severity, component_id, configs.deleted_at as config_deleted_at
FROM config_changes
    INNER JOIN config_component_relationships relations on relations.config_id = config_changes.config_id
    INNER JOIN config_items configs on configs.id = config_changes.config_id
ORDER BY
    config_changes.created_at DESC;

-- config_tags
DROP VIEW IF EXISTS config_tags;
CREATE OR REPLACE VIEW config_tags AS
  SELECT d.key, d.value
  FROM configs JOIN json_each_text(tags::json) d ON true GROUP BY d.key, d.value ORDER BY key, value;


-- config_type_summary
DROP VIEW IF EXISTS config_summary;
CREATE VIEW config_summary AS
  WITH changes_per_type AS (
    SELECT
      config_items.type,
      COUNT(config_changes.id) AS count
    FROM
      config_changes
      LEFT JOIN config_items ON config_changes.config_id = config_items.id
    WHERE config_changes.created_at > now() - interval '30 days'
    GROUP BY
      config_items.type
  ),
  analysis_counts AS (
    SELECT
      config_items.type,
      config_analysis.analysis_type,
      COUNT(*) AS count
    FROM
      config_analysis
      LEFT JOIN config_items ON config_items.id = config_analysis.config_id
    WHERE config_analysis.status = 'open'
    GROUP BY
      config_items.type,
      config_analysis.analysis_type
  ),
  aggregated_analysis_counts AS (
    SELECT
      type,
      json_object_agg(analysis_type, count) :: jsonb AS analysis
    FROM
      analysis_counts
    GROUP BY
      type
  ),
  analysis_by_severity AS (
    SELECT config_items.type, config_analysis.severity, COUNT(*) AS count
    FROM
      config_analysis
      LEFT JOIN config_items ON config_items.id = config_analysis.config_id
    WHERE
      config_analysis.status = 'open'
    GROUP BY
      config_items.type,
      config_analysis.severity
  ),
  aggregated_analysis_severity_counts AS (
    SELECT
      type,
      json_object_agg(severity, count) :: jsonb AS severity
    FROM
      analysis_by_severity
    GROUP BY
      type
  )
  SELECT
    config_items.type,
    MAX(config_items.created_at) as created_at,
    MAX(config_items.updated_at) as updated_at,
    aggregated_analysis_counts.analysis,
    aggregated_analysis_severity_counts.severity,
    changes_per_type.count AS changes,
    COUNT(*) AS total_configs,
    SUM(cost_per_minute) AS cost_per_minute,
    SUM(cost_total_1d) AS cost_total_1d,
    SUM(cost_total_7d) AS cost_total_7d,
    SUM(cost_total_30d) AS cost_total_30d
  FROM
    config_items
    LEFT JOIN changes_per_type ON config_items.type = changes_per_type.type
    LEFT JOIN aggregated_analysis_counts ON config_items.type = aggregated_analysis_counts.type
    LEFT JOIN aggregated_analysis_severity_counts ON config_items.type = aggregated_analysis_severity_counts.type
  GROUP BY
    config_items.type,
    changes_per_type.count,
    aggregated_analysis_counts.analysis,
    aggregated_analysis_severity_counts.severity
  ORDER BY
    type;

-- config_class_summary
DROP VIEW IF EXISTS config_class_summary;
CREATE VIEW config_class_summary AS
  WITH changes_per_type AS (
    SELECT
      config_items.config_class,
      COUNT(config_changes.id) AS count
    FROM
      config_changes
      LEFT JOIN config_items ON config_changes.config_id = config_items.id
    WHERE config_changes.created_at > now() - interval '30 days'
    GROUP BY
      config_items.config_class
  ),
  analysis_counts AS (
    SELECT
      config_items.config_class,
      config_analysis.analysis_type,
      COUNT(*) AS count
    FROM
      config_analysis
      LEFT JOIN config_items 
    ON config_items.id = config_analysis.config_id
    WHERE
      config_analysis.status = 'open'
    GROUP BY
      config_items.config_class,
      config_analysis.analysis_type
  ),
  aggregated_analysis_counts AS (
    SELECT
      config_class,
      json_object_agg(analysis_type, count) :: jsonb AS analysis
    FROM
      analysis_counts
    GROUP BY
      config_class
  )
  SELECT
    config_items.config_class,
    aggregated_analysis_counts.analysis,
    changes_per_type.count AS changes,
    COUNT(*) AS total_configs,
    SUM(cost_per_minute) AS cost_per_minute,
    SUM(cost_total_1d) AS cost_total_1d,
    SUM(cost_total_7d) AS cost_total_7d,
    SUM(cost_total_30d) AS cost_total_30d
  FROM
    config_items
    LEFT JOIN changes_per_type ON config_items.config_class = changes_per_type.config_class
    LEFT JOIN aggregated_analysis_counts ON config_items.config_class = aggregated_analysis_counts.config_class
  GROUP BY
    config_items.config_class,
    changes_per_type.count,
    aggregated_analysis_counts.analysis
  ORDER BY
    config_class;

DROP VIEW IF EXISTS config_analysis_analyzers;
CREATE OR REPLACE VIEW config_analysis_analyzers AS
  SELECT DISTINCT(analyzer) FROM config_analysis WHERE status = 'open';

DROP VIEW IF EXISTS config_changes_by_types;
CREATE OR REPLACE VIEW config_changes_by_types AS
  SELECT config_items.type, COUNT(config_changes.id) as change_count
  FROM config_changes
  INNER JOIN config_items ON config_changes.config_id = config_items.id 
  WHERE config_changes.created_at >= now() - INTERVAL '30 days'
  GROUP BY config_items.type
  ORDER BY change_count;

DROP VIEW IF EXISTS config_analysis_by_severity;
CREATE OR REPLACE VIEW config_analysis_by_severity AS
  SELECT config_items.type, config_analysis.severity, COUNT(*) as analysis_count
  FROM config_analysis
  INNER JOIN config_items ON config_analysis.config_id = config_items.id
  WHERE config_analysis.first_observed >= now() - INTERVAL '30 days'
  GROUP BY config_items.type, config_analysis.severity
  ORDER BY config_items.type, config_analysis.severity;

CREATE OR REPLACE FUNCTION insert_config_create_update_delete_in_event_queue()
RETURNS TRIGGER AS
$$
BEGIN
    DECLARE
      event_name TEXT;
    BEGIN
      CASE
        WHEN TG_OP = 'INSERT' THEN
          event_name := 'config.created';
        WHEN TG_OP = 'UPDATE' THEN
          event_name := 'config.updated';
          IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            event_name := 'config.deleted';
          END IF;
        ELSE
          RAISE EXCEPTION 'Unexpected operation in trigger: %', TG_OP;
      END CASE;
      
      INSERT INTO event_queue(name, properties)
      VALUES (event_name, jsonb_build_object('id', NEW.id))
      ON CONFLICT (name, properties) DO UPDATE
      SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    END;

    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_items_create_update_trigger
AFTER INSERT OR UPDATE
ON config_items
FOR EACH ROW
EXECUTE FUNCTION insert_config_create_update_delete_in_event_queue();

CREATE OR REPLACE VIEW config_detail AS
  SELECT
    ci.*,
    json_build_object(
      'relationships',  COALESCE(related.related_count, 0) + COALESCE(reverse_related.related_count, 0),
      'analysis', COALESCE(analysis.analysis_count, 0),
      'changes', COALESCE(config_changes.changes_count, 0),
      'playbook_runs', COALESCE(playbook_runs.playbook_runs_count, 0)
    ) as summary
  FROM config_items as ci
    LEFT JOIN
      (SELECT config_id, count(*) as related_count FROM config_relationships GROUP BY config_id) as related
      ON ci.id = related.config_id
    LEFT JOIN
      (SELECT related_id, count(*) as related_count FROM config_relationships GROUP BY related_id) as reverse_related
      ON ci.id = reverse_related.related_id
    LEFT JOIN
      (SELECT config_id, count(*) as analysis_count FROM config_analysis GROUP BY config_id) as analysis
      ON ci.id = analysis.config_id
    LEFT JOIN 
      (SELECT config_id, count(*) as changes_count FROM config_changes GROUP BY config_id) as config_changes 
      ON ci.id = config_changes.config_id
    LEFT JOIN 
      (SELECT config_id, count(*) as playbook_runs_count FROM playbook_runs GROUP BY config_id) as playbook_runs
      ON ci.id = playbook_runs.config_id;

CREATE OR REPLACE VIEW config_analysis_items AS
  SELECT
    ca.*,
    ci.type as config_type,
    ci.config_class
  FROM config_analysis as ca
    LEFT JOIN config_items as ci ON ca.config_id = ci.id;

CREATE OR REPLACE VIEW config_changes_items AS
  SELECT
    cc.*,
    ci.type as config_type,
    ci.config_class
  FROM config_changes as cc
    LEFT JOIN config_items as ci ON cc.config_id = ci.id;


-- related config ids
DROP FUNCTION IF EXISTS related_config_ids(UUID, TEXT, BOOLEAN);

CREATE FUNCTION related_config_ids (
  config_id UUID,
  type_filter TEXT DEFAULT 'all',
  include_deleted_configs BOOLEAN DEFAULT false
)
RETURNS TABLE (
  relation TEXT,
  relation_type TEXT,
  id UUID
) AS $$
BEGIN
  RETURN query
    SELECT
      config_relationships.relation,
      'outgoing' AS relation_type,
      c.id
    FROM config_relationships
      INNER JOIN configs AS c ON config_relationships.related_id = c.id AND (related_config_ids.include_deleted_configs OR c.deleted_at IS NULL)
    WHERE
      config_relationships.deleted_at IS NULL
      AND config_relationships.config_id = related_config_ids.config_id
      AND (related_config_ids.type_filter = 'outgoing' OR related_config_ids.type_filter = 'all')
    UNION
    SELECT
      config_relationships.relation,
      'incoming' AS relation_type,
      c.id
    FROM config_relationships
      INNER JOIN configs AS c ON config_relationships.config_id = c.id AND (related_config_ids.include_deleted_configs OR c.deleted_at IS NULL)
    WHERE
      config_relationships.deleted_at IS NULL
      AND config_relationships.related_id = related_config_ids.config_id
      AND (related_config_ids.type_filter = 'incoming' OR related_config_ids.type_filter = 'all');
END;
$$ LANGUAGE plpgsql;

-- related configs
DROP FUNCTION IF EXISTS related_configs(UUID, TEXT, BOOLEAN);

CREATE FUNCTION related_configs (
  config_id UUID,
  type_filter TEXT DEFAULT 'all',
  include_deleted_configs BOOLEAN DEFAULT FALSE
)
RETURNS TABLE (
  relation TEXT,
  relation_type TEXT,
  config JSONB
) AS $$
BEGIN
  RETURN query
    SELECT
      r.relation,
      r.relation_type,
      jsonb_build_object(
        'id', c.id,
        'name', c.name, 
        'type', c.type, 
        'tags', c.tags, 
        'changes', c.changes,
        'analysis', c.analysis,
        'cost_per_minute', c.cost_per_minute,
        'cost_total_1d', c.cost_total_1d,
        'cost_total_7d', c.cost_total_7d,
        'cost_total_30d', c.cost_total_30d,
        'created_at', c.created_at, 
        'updated_at', c.updated_at
      ) AS config
    FROM related_config_ids($1, $2, $3) as r
    LEFT JOIN configs AS c ON r.id = c.id;
END;
$$ LANGUAGE plpgsql;
