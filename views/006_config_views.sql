-- dependsOn: functions/drop.sql

-- Add cascade drops first to make sure all functions and views are always recreated
DROP VIEW IF EXISTS configs CASCADE;

DROP FUNCTION IF EXISTS related_changes_recursive CASCADE;

CREATE MATERIALIZED VIEW IF NOT EXISTS
  config_item_summary_3d AS
WITH type_counts AS (
    SELECT
        ca.config_id,
        ca.analysis_type,
        COUNT(*) AS type_count
    FROM
        config_analysis ca
    WHERE
        ca.status = 'open'
    GROUP BY
        ca.config_id, ca.analysis_type
)
SELECT
    ci.id AS config_id,
    COUNT(cc.config_id) AS config_changes_count,
    COALESCE(
        (SELECT jsonb_object_agg(tc.analysis_type, tc.type_count)
         FROM type_counts tc
         WHERE tc.config_id = ci.id), '{}'::jsonb
    ) AS config_analysis_type_counts
FROM
    config_items ci
LEFT JOIN
    config_changes cc ON ci.id = cc.config_id AND cc.created_at >= NOW() - INTERVAL '3 days'
GROUP BY
    ci.id, ci.name;

CREATE OR REPLACE FUNCTION refresh_config_item_summary_3d() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW  config_item_summary_3d;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;


CREATE MATERIALIZED VIEW IF NOT EXISTS  config_item_summary_7d AS
WITH type_counts AS (
    SELECT
        ca.config_id,
        ca.analysis_type,
        COUNT(*) AS type_count
    FROM
        config_analysis ca
    WHERE
        ca.status = 'open'
    GROUP BY
        ca.config_id, ca.analysis_type
)
SELECT
    ci.id AS config_id,
    COUNT(cc.config_id) AS config_changes_count,
    COALESCE(
        (SELECT jsonb_object_agg(tc.analysis_type, tc.type_count)
         FROM type_counts tc
         WHERE tc.config_id = ci.id), '{}'::jsonb
    ) AS config_analysis_type_counts
FROM
    config_items ci
LEFT JOIN
    config_changes cc ON ci.id = cc.config_id AND cc.created_at >= NOW() - INTERVAL '7 days'
GROUP BY
    ci.id, ci.name;


CREATE OR REPLACE FUNCTION refresh_config_item_summary_7d() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW  config_item_summary_7d;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE MATERIALIZED VIEW IF NOT EXISTS config_item_summary_30d AS
WITH type_counts AS (
    SELECT
        ca.config_id,
        ca.analysis_type,
        COUNT(*) AS type_count
    FROM
        config_analysis ca
    WHERE
        ca.status = 'open'
    GROUP BY
        ca.config_id, ca.analysis_type
)
SELECT
    ci.id AS config_id,
    COUNT(cc.config_id) AS config_changes_count,
    COALESCE(
        (SELECT jsonb_object_agg(tc.analysis_type, tc.type_count)
         FROM type_counts tc
         WHERE tc.config_id = ci.id), '{}'::jsonb
    ) AS config_analysis_type_counts
FROM
    config_items ci
LEFT JOIN
    config_changes cc ON ci.id = cc.config_id AND cc.created_at >= NOW() - INTERVAL '30 days'
GROUP BY
    ci.id, ci.name;



CREATE OR REPLACE FUNCTION refresh_config_item_summary_30d() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW  config_item_summary_30d;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE or REPLACE VIEW configs AS
  SELECT
    ci.id,
    ci.scraper_id,
    ci.config_class,
    ci.external_id,
    ci.type,
    ci.name,
    ci.tags->>'namespace' as namespace,
    ci.description,
    ci.source,
    ci.labels,
    ci.tags,
    ci.tags_values,
    ci.properties,
    ci.properties_values,
    ci.created_by,
    ci.created_at,
    ci.updated_at,
    ci.deleted_at,
    ci.cost_per_minute,
    ci.cost_total_1d,
    ci.cost_total_7d,
    ci.cost_total_30d,
    ci.agent_id,
    ci.status,
    ci.health,
    ci.ready,
    ci.path,
    config_item_summary_7d.config_changes_count AS changes,
    config_item_summary_7d.config_analysis_type_counts AS analysis
  FROM config_items AS ci
  LEFT JOIN config_item_summary_7d ON config_item_summary_7d.config_id = ci.id;


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
  FROM config_items w JOIN json_each_text(tags::json) d ON true where deleted_at is null GROUP BY d.key, d.value ORDER BY key, value;


-- config_labels
DROP VIEW IF EXISTS config_labels;
CREATE OR REPLACE VIEW config_labels AS
  SELECT d.key, d.value
  FROM config_items w JOIN json_each_text(labels::json) d ON true where deleted_at is null GROUP BY d.key, d.value ORDER BY key, value;


-- config_tags_labels_keys
DROP VIEW IF EXISTS config_tags_labels_keys;
CREATE OR REPLACE VIEW config_tags_labels_keys AS
  SELECT DISTINCT 'tag:' || jsonb_object_keys(tags) AS "key" FROM config_items
  UNION
  SELECT DISTINCT 'label:' || jsonb_object_keys(labels) AS "key" FROM config_items;

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
  WHERE config_items.deleted_at IS NULL
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
          IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            event_name := 'config.deleted';
          ELSE
            RETURN NEW;
          END IF;
        ELSE
          RAISE EXCEPTION 'Unexpected operation in trigger: %', TG_OP;
      END CASE;

      INSERT INTO event_queue(name, properties)
      VALUES (event_name, jsonb_build_object('id', NEW.id))
      ON CONFLICT (name, md5(properties::text)) DO UPDATE
      SET created_at = NOW(), last_attempt = NULL, attempts = 0;
    END;

    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_items_create_update_trigger
AFTER INSERT OR UPDATE ON config_items
FOR EACH ROW
  EXECUTE FUNCTION insert_config_create_update_delete_in_event_queue();

---
CREATE OR REPLACE FUNCTION insert_config_changes_updates_in_event_queue()
RETURNS TRIGGER AS
$$
DECLARE
  event_name TEXT := 'config.changed';
BEGIN
  IF NEW.change_type = 'diff' THEN
    event_name := 'config.updated';
  END IF;

  INSERT INTO event_queue(name, properties)
  VALUES (event_name, jsonb_build_object('id', NEW.config_id, 'change_id', NEW.id))
  ON CONFLICT (name, md5(properties::text)) DO UPDATE
  SET created_at = NOW(), last_attempt = NULL, attempts = 0;

  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_change_create_update_trigger
AFTER INSERT OR UPDATE ON config_changes
FOR EACH ROW
  EXECUTE FUNCTION insert_config_changes_updates_in_event_queue();
---

CREATE OR REPLACE VIEW config_analysis_items AS
  SELECT
    ca.*,
    ci.type as config_type,
    ci.config_class
  FROM config_analysis as ca
    LEFT JOIN config_items as ci ON ca.config_id = ci.id;

-- related_config_ids_recursive---
DROP FUNCTION IF EXISTS related_config_ids_recursive;

CREATE OR REPLACE FUNCTION related_config_ids_recursive (
  config_id UUID,
  type_filter TEXT DEFAULT 'outgoing',
  max_depth INT DEFAULT 5,
  incoming_relation TEXT DEFAULT 'both', -- hard or both (hard & soft)
  outgoing_relation TEXT DEFAULT 'both' -- hard or both (hard & soft)
) RETURNS TABLE (id UUID, direction TEXT, depth INT) AS $$
BEGIN

RETURN query
  WITH edges as (
    SELECT * FROM config_relationships_recursive(config_id, type_filter, max_depth, incoming_relation, outgoing_relation)
  ), all_ids AS (
    SELECT edges.id, edges.depth, edges.direction FROM edges
    UNION
    SELECT edges.related_id as id, edges.depth, edges.direction FROM edges
  ) SELECT all_ids.id, all_ids.direction, MIN(all_ids.depth) depth FROM all_ids
    GROUP BY all_ids.id, all_ids.direction
    ORDER BY depth;
  END;

$$ LANGUAGE plpgsql;

-- config_relationships_recursive --
DROP FUNCTION IF EXISTS config_relationships_recursive;

CREATE OR REPLACE FUNCTION config_relationships_recursive (
  config_id UUID,
  type_filter TEXT DEFAULT 'outgoing',
  max_depth INT DEFAULT 5,
  incoming_relation TEXT DEFAULT 'both', -- hard or both (hard & soft)
  outgoing_relation TEXT DEFAULT 'both' -- hard or both (hard & soft)
) RETURNS TABLE (id UUID, related_id UUID, relation_type TEXT, direction TEXT, depth INT) AS $$
  BEGIN

  IF type_filter NOT IN ('incoming', 'outgoing', 'all', '', null) THEN
    RAISE EXCEPTION 'Invalid type_filter value. Allowed values are: ''incoming'', ''outgoing'', ''all''';
  END IF;

IF type_filter = 'outgoing' THEN
  RETURN query
      WITH RECURSIVE cte (config_id, related_id, relation, direction, depth) AS (
        SELECT parent.config_id, parent.related_id, parent.relation, 'outgoing', 1::int
        FROM config_relationships parent
        WHERE parent.config_id = config_relationships_recursive.config_id
          AND (outgoing_relation = 'both' OR incoming_relation = 'soft'  OR (outgoing_relation = 'hard' AND parent.relation = 'hard'))
          AND deleted_at IS NULL
        UNION ALL
        SELECT
          parent.related_id as config_id, child.related_id, child.relation, 'outgoing', parent.depth + 1
          FROM config_relationships child, cte parent
          WHERE child.config_id = parent.related_id
            AND parent.depth < max_depth
            AND (outgoing_relation = 'both' OR incoming_relation = 'soft'  OR (outgoing_relation = 'hard' AND child.relation = 'hard'))
            AND deleted_at IS NULL
      ) CYCLE config_id SET is_cycle USING path
      SELECT DISTINCT cte.config_id, cte.related_id, cte.relation as "relation_type", type_filter as "direction", cte.depth
      FROM cte
      ORDER BY cte.depth asc;
ELSIF type_filter = 'incoming' THEN
  RETURN query
      WITH RECURSIVE cte (config_id, related_id, relation, direction, depth) AS (
        SELECT parent.config_id, parent.related_id as related_id, parent.relation, 'incoming', 1::int
        FROM config_relationships parent
        WHERE parent.related_id = config_relationships_recursive.config_id
          AND (incoming_relation = 'both' OR  incoming_relation = 'soft' OR (incoming_relation = 'hard' AND parent.relation = 'hard'))
          AND deleted_at IS NULL
        UNION ALL
        SELECT
          child.config_id, child.related_id as related_id, child.relation, 'incoming', parent.depth + 1
          FROM config_relationships child, cte parent
          WHERE child.related_id = parent.config_id
            AND parent.depth < max_depth
            AND (incoming_relation = 'both' OR incoming_relation = 'soft' OR (incoming_relation = 'hard' AND child.relation = 'hard'))
            AND deleted_at IS NULL
      ) CYCLE config_id SET is_cycle USING path
      SELECT DISTINCT cte.config_id, cte.related_id, cte.relation AS "relation_type", type_filter as "direction", cte.depth
      FROM cte
      ORDER BY cte.depth asc;
ELSE
  RETURN query
      SELECT * FROM config_relationships_recursive(config_id, 'incoming', max_depth, incoming_relation, outgoing_relation)
      UNION
      SELECT * FROM config_relationships_recursive(config_id, 'outgoing', max_depth, incoming_relation, outgoing_relation);
END IF;

  END;
$$ LANGUAGE plpgsql;

DROP FUNCTION IF EXISTS get_recursive_path;

CREATE OR REPLACE FUNCTION get_recursive_path(start uuid)
RETURNS TABLE  (id UUID, related_id UUID, relation_type TEXT, direction TEXT, depth INT) AS $$
DECLARE
    current_id uuid;
    current_parent uuid;
    current_depth INT;
    current_path text[];
BEGIN
    -- Initialize the starting point
    current_id := start;
    current_parent := NULL;
    current_depth := 0;

    select string_to_array(config_items.path, '.') into current_path from config_items where config_items.id = start;

    IF array_length(current_path, 1) > 0  THEN
      FOR i IN 0 .. array_length(current_path, 1) -1 LOOP
          current_parent := current_id;
          current_id := current_path[array_length(current_path,1) - i];

      if start != current_id then
            current_depth := current_depth + 1;
            RETURN QUERY SELECT current_id,current_parent,'parent','incoming', current_depth;
          end if;
      END LOOP;
    END IF;

END;
$$ LANGUAGE plpgsql;



CREATE OR REPLACE FUNCTION drop_config_items(ids text[]) RETURNS void as $$
  BEGIN
  ALTER TABLE config_items
    ALTER CONSTRAINT config_items_parent_id_fkey DEFERRABLE INITIALLY DEFERRED;
  SET CONSTRAINTS ALL DEFERRED;
  DELETE FROM config_component_relationships WHERE config_id  = any (ids::uuid[]);
  DELETE FROM check_config_relationships WHERE config_id  = any  (ids::uuid[]);
  DELETE FROM config_changes WHERE config_id   = any  (ids::uuid[]);
  DELETE FROM config_relationships WHERE config_id  = any  (ids::uuid[]) or related_id  = any(ids::uuid[]);
  DELETE FROM config_analysis WHERE config_id   = any  (ids::uuid[]);

      FOR i IN 1 .. array_length(ids, 1) LOOP
       DELETE FROM config_items WHERE PATH like '%'||ids[i]||'%';
      END LOOP;

  DELETE FROM config_items WHERE parent_id  = any (ids::uuid[]);
  DELETE FROM config_items WHERE id  = any (ids::uuid[]);
  END;
$$ LANGUAGE plpgsql;

-- related configs recursively
DROP FUNCTION IF EXISTS related_configs_recursive;
CREATE FUNCTION related_configs_recursive (
  config_id UUID,
  type_filter TEXT DEFAULT 'outgoing',
  include_deleted_configs BOOLEAN DEFAULT FALSE,
  max_depth INTEGER DEFAULT 5,
  incoming_relation TEXT DEFAULT 'both', -- hard or both (hard & soft)
  outgoing_relation TEXT DEFAULT 'both' -- hard or both (hard & soft)
) RETURNS TABLE (
    id uuid,
    name TEXT,
    type TEXT,
    related_ids TEXT[],
    tags jsonb,
    changes BIGINT,
    analysis jsonb,
    cost_per_minute NUMERIC(16, 4),
    cost_total_1d NUMERIC(16, 4),
    cost_total_7d NUMERIC(16, 4),
    cost_total_30d NUMERIC(16, 4),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    agent_id uuid,
    health TEXT,
    ready BOOLEAN,
    status TEXT,
    path TEXT
) AS $$
BEGIN
  RETURN query
    WITH edges as (
      SELECT * FROM config_relationships_recursive(config_id, type_filter, max_depth, incoming_relation, outgoing_relation)
      UNION
      SELECT * from get_recursive_path(config_id)
    ),
     all_ids AS (
      SELECT edges.id FROM edges
      UNION
      SELECT edges.related_id as id FROM edges WHERE max_depth > 0
      UNION
      SELECT related_configs_recursive.config_id as id
    ), grouped_related_ids AS (
      SELECT all_ids.id, MIN(edges.depth) depth, array_agg(DISTINCT edges.related_id::TEXT) FILTER (WHERE edges.related_id IS NOT NULL) as related_ids
      FROM all_ids
      LEFT JOIN edges ON edges.id = all_ids.id
      GROUP BY all_ids.id
    )
      SELECT
        configs.id,
        configs.name,
        configs.type,
        grouped_related_ids.related_ids,
        configs.tags,
        configs.changes,
        configs.analysis,
        configs.cost_per_minute,
        configs.cost_total_1d,
        configs.cost_total_7d,
        configs.cost_total_30d,
        configs.created_at,
        configs.updated_at,
        configs.deleted_at,
        configs.agent_id,
        configs.health,
        configs.ready,
        configs.status,
        configs.path
      FROM configs
      LEFT JOIN grouped_related_ids ON configs.id = grouped_related_ids.id
      WHERE configs.id IN (SELECT DISTINCT all_ids.id FROM all_ids)
      AND (include_deleted_configs OR configs.deleted_at IS NULL)
      ORDER BY grouped_related_ids.depth;
END;
$$ LANGUAGE plpgsql;

-- related configs
DROP FUNCTION IF EXISTS related_configs(config_id uuid, include_deleted_configs boolean);
DROP FUNCTION IF EXISTS related_configs(config_id uuid, type_filter text, include_deleted_configs boolean);

CREATE FUNCTION related_configs (
  config_id UUID,
  type_filter TEXT DEFAULT 'all',
  include_deleted_configs BOOLEAN DEFAULT FALSE
) RETURNS TABLE (
    id uuid,
    name TEXT,
    type TEXT,
    related_ids TEXT[],
    tags jsonb,
    changes json,
    analysis json,
    cost_per_minute NUMERIC(16, 4),
    cost_total_1d NUMERIC(16, 4),
    cost_total_7d NUMERIC(16, 4),
    cost_total_30d NUMERIC(16, 4),
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    agent_id uuid,
    health TEXT,
    ready BOOLEAN,
    status TEXT,
    path TEXT
) AS $$
BEGIN
  RETURN query
    SELECT * from related_configs_recursive(config_id, type_filter, include_deleted_configs, 1);
END;
$$ LANGUAGE plpgsql;

-- related config changes recursively
CREATE OR REPLACE FUNCTION related_changes_recursive (
  lookup_id UUID,
  type_filter TEXT DEFAULT 'downstream',  -- 'downstream', 'upstream', 'all', 'none' or ''
  include_deleted_configs BOOLEAN DEFAULT FALSE,
  max_depth INTEGER DEFAULT 5,
  soft BOOLEAN DEFAULT FALSE
) RETURNS TABLE (
    id uuid,
    config_id uuid,
    name TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE,
    type TEXT,
    tags jsonb,
    external_created_by TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    severity TEXT,
    change_type TEXT,
    source TEXT,
    summary TEXT,
    created_by uuid,
    count INT ,
    first_observed TIMESTAMP WITH TIME ZONE,
    agent_id uuid
) AS $$
BEGIN
  IF type_filter NOT IN ('upstream', 'downstream', 'all', 'none', '',null) THEN
    RAISE EXCEPTION 'Invalid type_filter value. Allowed values are: ''upstream'', ''downstream'', ''all'', ''none'' or ''''';
  END IF;

  IF type_filter IN ('none', '',null) THEN
    RETURN query
      SELECT
          cc.id, cc.config_id, config_items.name, config_items.deleted_at, config_items.type, config_items.tags, cc.external_created_by,
          cc.created_at, cc.severity, cc.change_type, cc.source, cc.summary, cc.created_by, cc.count, cc.first_observed, config_items.agent_id
      FROM config_changes cc
      LEFT JOIN config_items on config_items.id = cc.config_id
      WHERE cc.config_id = lookup_id;

  ELSIF type_filter IN ('downstream') THEN
    RETURN query
      SELECT DISTINCT ON (cc.id)
          cc.id, cc.config_id, config_items.name, config_items.deleted_at, config_items.type, config_items.tags, cc.external_created_by,
          cc.created_at, cc.severity, cc.change_type, cc.source, cc.summary, cc.created_by, cc.count, cc.first_observed, config_items.agent_id
      FROM config_changes cc
      LEFT JOIN config_items on config_items.id = cc.config_id
      LEFT JOIN
          (SELECT config_relationships.config_id, config_relationships.related_id
           FROM config_relationships
           WHERE relation != 'hard') AS cr
           ON (cr.config_id = cc.config_id OR (soft AND cr.related_id = cc.config_id))
      WHERE config_items.path LIKE (
        SELECT CASE
            WHEN config_items.path = '' THEN config_items.id::text
            ELSE CONCAT(config_items.path, '.', config_items.id)
          END
        FROM config_items WHERE config_items.id = lookup_id
        ) || '%' OR
        (cc.config_id = lookup_id) OR
        (soft AND (cr.config_id = lookup_id OR cr.related_id = lookup_id));

  ELSIF type_filter IN ('upstream') THEN
    RETURN query
      SELECT DISTINCT ON (cc.id)
          cc.id, cc.config_id, config_items.name, config_items.deleted_at, config_items.type, config_items.tags, cc.external_created_by,
          cc.created_at, cc.severity, cc.change_type, cc.source, cc.summary, cc.created_by, cc.count, cc.first_observed, config_items.agent_id
      FROM config_changes cc
      LEFT JOIN config_items on config_items.id = cc.config_id
      LEFT JOIN
          (SELECT config_relationships.config_id, config_relationships.related_id
           FROM config_relationships
           WHERE relation != 'hard') AS cr
           ON (cr.config_id = cc.config_id OR (soft AND cr.related_id = cc.config_id))
      WHERE cc.config_id IN (SELECT get_recursive_path.id FROM get_recursive_path(lookup_id)) OR
        (cc.config_id = lookup_id) OR
        (soft AND (cr.config_id = lookup_id OR cr.related_id = lookup_id));

  ELSE
    RETURN query
      SELECT
          cc.id, cc.config_id, c.name, c.deleted_at, c.type, c.tags, cc.external_created_by,
          cc.created_at, cc.severity, cc.change_type, cc.source, cc.summary, cc.created_by, cc.count, cc.first_observed, c.agent_id
      FROM config_changes cc
      LEFT JOIN config_items c on c.id = cc.config_id
      WHERE cc.config_id = lookup_id
        OR cc.config_id IN (
          SELECT related_config_ids_recursive.id
          FROM related_config_ids_recursive(
            lookup_id,
            CASE
              WHEN type_filter = 'upstream' THEN 'incoming'
              ELSE type_filter
            END,
            max_depth
          )
        );
  END IF;
END;
$$ LANGUAGE plpgsql;

DROP VIEW IF EXISTS catalog_changes;

CREATE OR REPLACE VIEW catalog_changes AS
  SELECT
    cc.id,
    cc.config_id,
    c.name,
    c.deleted_at,
    c.type,
    c.tags,
    c.config,
    cc.external_created_by,
    cc.created_at,
    cc.severity,
    cc.change_type,
    cc.source,
    cc.summary,
    cc.details,
    cc.diff,
    cc.created_by,
    cc.count,
    cc.first_observed,
    c.agent_id
  FROM config_changes cc
  LEFT JOIN config_items c on c.id = cc.config_id;

DROP VIEW IF EXISTS config_detail;

CREATE OR REPLACE VIEW config_detail AS
  SELECT
    ci.*,
    config_items_last_scraped_time.last_scraped_time,
    agents.name as agent_name,
    json_build_object(
      'relationships',  COALESCE(related.related_count, 0) + COALESCE(reverse_related.related_count, 0),
      'analysis', COALESCE(analysis.analysis_count, 0),
      'changes', COALESCE(change_summary.total_changes_count, 0),
      'playbook_runs', COALESCE(playbook_runs.playbook_runs_count, 0),
      'checks', COALESCE(config_checks.checks_count, 0)
    ) as summary,
    CASE WHEN config_scrapers.id IS NOT NULL THEN json_build_object(
      'id', config_scrapers.id,
      'name', config_scrapers.name
    ) ELSE NULL END as scraper
  FROM config_items as ci
    LEFT JOIN agents ON agents.id = ci.agent_id
    LEFT JOIN config_items_last_scraped_time ON config_items_last_scraped_time.config_id = ci.id
    LEFT JOIN config_scrapers ON config_scrapers.id = ci.scraper_id
    LEFT JOIN
      (SELECT config_id, count(*) as related_count FROM config_relationships GROUP BY config_id) as related
      ON ci.id = related.config_id
    LEFT JOIN
      (SELECT related_id, count(*) as related_count FROM config_relationships GROUP BY related_id) as reverse_related
      ON ci.id = reverse_related.related_id
    LEFT JOIN
      (SELECT config_id, SUM(value::INT) as analysis_count FROM config_item_summary_7d
       CROSS JOIN LATERAL jsonb_each_text(config_analysis_type_counts)
        GROUP BY config_id) as analysis
      ON ci.id = analysis.config_id
    LEFT JOIN
      (SELECT ci.id AS config_id, SUM(cs.config_changes_count) AS total_changes_count
        FROM config_items ci
        LEFT JOIN config_item_summary_7d cs ON ci.path LIKE '%' || cs.config_id || '%'
        GROUP BY ci.id) AS change_summary
      ON ci.id = change_summary.config_id
    LEFT JOIN
      (SELECT config_id, count(*) as playbook_runs_count FROM playbook_runs
        WHERE start_time > NOW() - interval '30 days'
        GROUP BY config_id) as playbook_runs
      ON ci.id = playbook_runs.config_id
    LEFT JOIN
      (SELECT config_id, count(*) as checks_count from check_config_relationships
        WHERE deleted_at IS NULL
        GROUP BY config_id) as config_checks
      ON ci.id = config_checks.config_id
    LEFT JOIN
      (SELECT
          config_id, json_agg(components) as components
        FROM
          (SELECT
              ccr.config_id as config_id, components
            FROM config_component_relationships as ccr
            LEFT JOIN components ON components.id = ccr.component_id
          ) as config_components
        GROUP BY config_id) as gcc
        ON ci.id = gcc.config_id;

--- config_path is a function that given a config id returns its path by walking the tree recursively up using the parent id and then joining the ids with a `.`
CREATE OR REPLACE FUNCTION config_path(UUID)
RETURNS TEXT AS $$
DECLARE
    child_id UUID;
    parent_id UUID;
    parent_path TEXT;
BEGIN
    SELECT config_items.id, config_items.parent_id INTO child_id, parent_id FROM config_items WHERE config_items.id = $1;

    IF parent_id IS NULL THEN
        RETURN child_id;
    ELSE
        SELECT config_path(parent_id) INTO parent_path;
        RETURN parent_path || '.' || child_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- config parent to config_relationship trigger
CREATE OR REPLACE FUNCTION insert_parent_to_config_relationship()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.parent_id IS NOT NULL THEN
      INSERT INTO config_relationships (config_id, related_id, relation)
      VALUES (NEW.parent_id, NEW.id, 'hard')
      ON CONFLICT DO NOTHING;
    END IF;
  ELSIF TG_OP = 'UPDATE' THEN
    IF NEW.parent_id IS DISTINCT FROM OLD.parent_id THEN
      IF OLD.parent_id IS NOT NULL THEN
        DELETE FROM config_relationships
        WHERE config_id = OLD.parent_id AND related_id = NEW.id AND relation = 'hard';
      END IF;

      IF NEW.parent_id IS NOT NULL THEN
        INSERT INTO config_relationships (config_id, related_id, relation)
        VALUES (NEW.parent_id, NEW.id, 'hard')
        ON CONFLICT DO NOTHING;
      END IF;
    END IF;
  END IF;

  RETURN NULL;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER fn_insert_parent_to_config_relationship
AFTER INSERT OR UPDATE ON config_items
FOR EACH ROW
EXECUTE PROCEDURE insert_parent_to_config_relationship();

DROP VIEW IF EXISTS config_statuses;
CREATE or REPLACE VIEW config_statuses AS
  SELECT DISTINCT status FROM config_items WHERE status IS NOT NULL ORDER BY status;

DROP VIEW IF EXISTS check_summary_by_config;
DROP VIEW IF EXISTS checks_by_config;

-- checks_by_config
CREATE
OR REPLACE VIEW checks_by_config AS
SELECT
  check_config_relationships.config_id,
  checks.id,
  checks.type,
  checks.name,
  checks.severity,
  checks.status,
  checks.icon,
  checks_unlogged.last_runtime
FROM
  check_config_relationships
  INNER JOIN checks ON checks.id = check_config_relationships.check_id
  LEFT JOIN checks_unlogged ON checks.id = checks_unlogged.check_id
WHERE
  check_config_relationships.deleted_at IS NULL;

-- check_summary_by_config
CREATE OR REPLACE VIEW
  check_summary_by_config AS
WITH cte as (
    SELECT
        config_id, status, COUNT(*) AS count
    FROM
      checks_by_config
    GROUP BY
      config_id, status
)
SELECT config_id, json_object_agg(status, count) AS checks
FROM cte GROUP BY config_id;

-- When a new item is inserted, or aliases are updated,
-- we find the same alias for a different type and link them
-- Assumes (type, external_id) tuples are unique across the table
CREATE OR REPLACE FUNCTION create_alias_config_relationships_for_config_item(config_item_id UUID)
RETURNS VOID AS $$
DECLARE
    v_config_item RECORD;
BEGIN
    -- Get the config_item record
    SELECT id, type, external_id
    INTO v_config_item
    FROM config_items
    WHERE id = config_item_id
    AND deleted_at IS NULL;

    -- Check if record exists
    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Only proceed if external_id array is not null and not empty
    IF v_config_item.external_id IS NOT NULL AND array_length(v_config_item.external_id, 1) > 0 THEN
        INSERT INTO config_relationships (config_id, related_id, relation)
        SELECT
            v_config_item.id as config_id,
            ci.id as related_id,
            'Alias' as relation
        FROM config_items ci,
             unnest(v_config_item.external_id) as ext_id
        WHERE
            -- Find config_items that contain the same external_id and different type
            ci.external_id @> ARRAY[ext_id]
            AND ci.type != v_config_item.type
            AND ci.deleted_at IS NULL
            AND ci.id != v_config_item.id  -- Don't create relationship with itself
        ON CONFLICT (related_id, config_id, relation)
        DO NOTHING;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_alias_config_relationships()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM create_alias_config_relationships_for_config_item(NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER config_items_create_alias_relationships_insert
    AFTER INSERT ON config_items
    FOR EACH ROW
    EXECUTE FUNCTION create_alias_config_relationships();

-- Only fire when external_id or type changes to avoid unnecessary executions
CREATE OR REPLACE TRIGGER config_items_create_alias_relationships_update
    AFTER UPDATE OF external_id, type ON config_items
    FOR EACH ROW
    WHEN (OLD.external_id IS DISTINCT FROM NEW.external_id OR OLD.type IS DISTINCT FROM NEW.type)
    EXECUTE FUNCTION create_alias_config_relationships();

-- Function to update config_items_last_scraped_time when config_item is inserted
CREATE OR REPLACE FUNCTION update_config_items_last_scraped_time()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO config_items_last_scraped_time (config_id, last_scraped_time)
    VALUES (NEW.id, NOW())
    ON CONFLICT (config_id)
    DO UPDATE SET last_scraped_time = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to call the function after config_item insert
CREATE OR REPLACE TRIGGER config_items_update_last_scraped_time
    AFTER INSERT ON config_items
    FOR EACH ROW
    EXECUTE FUNCTION update_config_items_last_scraped_time();
