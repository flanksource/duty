DROP FUNCTION IF EXISTS lookup_component_names;

CREATE
OR REPLACE FUNCTION lookup_component_names (component_id uuid[]) RETURNS TABLE (names text[]) AS $$
BEGIN
    RETURN QUERY
        SELECT array_agg(name) FROM components where id = any( component_id);

END;
$$ LANGUAGE plpgsql;

-- Drop these first because of dependencies
DROP VIEW IF EXISTS topology;

DROP VIEW IF EXISTS check_summary_by_component;

DROP VIEW IF EXISTS checks_by_component;
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

-- analysis_by_config
DROP VIEW IF EXISTS analysis_by_config;
CREATE OR REPLACE VIEW
  analysis_by_config AS
WITH
  type_summary AS (
    SELECT
      summary.id,
      summary.type,
      json_object_agg(f.k, f.v) as json
    FROM
      (
        SELECT
          config_analysis.config_id AS id,
          analysis_type as
        type,
        json_build_object(severity, count(*)) AS severity_agg
        FROM
          config_analysis
        WHERE
          status != 'resolved'
        GROUP BY
          severity,
          analysis_type,
          config_id
      ) AS summary,
      json_each(summary.severity_agg) AS f (k, v)
    GROUP BY
      summary.type,
      summary.id
  )
SELECT
  id,
  jsonb_object_agg(key, value) as analysis
FROM
  (
    SELECT
      id,
      json_object_agg(
        type,
        json
      ) analysis
    from
      type_summary
    group by
      id,
    type
  ) i,
  json_each(analysis)
GROUP BY
  id;

-- analysis_by_component
DROP VIEW IF EXISTS analysis_by_component;

CREATE OR REPLACE VIEW
  analysis_by_component AS
SELECT
  config_analysis.config_id,
  configs.name,
  configs.config_class,
  configs.type,
  analysis_type,
  config_analysis.first_observed,
  config_analysis.last_observed,
  config_analysis.created_by,
  config_analysis.id as analysis_id,
  config_analysis.severity,
  component_id
FROM
  config_analysis
  INNER JOIN config_component_relationships relations on relations.config_id = config_analysis.config_id
  INNER JOIN config_items configs on configs.id = config_analysis.config_id
WHERE
  configs.deleted_at IS NULL AND config_analysis.status = 'open'
ORDER BY
    ARRAY_POSITION(ARRAY['critical', 'blocker', 'high', 'medium', 'low', 'info'], config_analysis.severity),
    configs.name;

-- analysis_summary_by_component
DROP VIEW IF EXISTS analysis_summary_by_component CASCADE;
CREATE OR REPLACE VIEW
  analysis_summary_by_component AS
WITH
  type_summary AS (
    SELECT
      summary.id,
      summary.type,
      json_object_agg(f.k, f.v) as json
    FROM
      (
        SELECT
          config_component_relationships.component_id AS id,
          config_analysis.analysis_type AS
        type,
        json_build_object(severity, count(*)) AS severity_agg
        FROM
          config_analysis
          LEFT JOIN config_component_relationships ON config_analysis.config_id = config_component_relationships.config_id
          INNER JOIN config_items configs ON configs.id = config_analysis.config_id
        WHERE
          config_component_relationships.deleted_at IS NULL
          AND configs.deleted_at IS NULL
        GROUP BY
          config_analysis.severity,
          config_analysis.analysis_type,
          config_component_relationships.component_id
      ) AS summary,
      json_each(summary.severity_agg) AS f (k, v)
    GROUP BY
      summary.type,
      summary.id
  )
SELECT
  id,
  jsonb_object_agg(key, value) AS analysis
FROM
  (
    SELECT
      id,
      json_object_agg(
        type,
        json
      ) analysis
    FROM
      type_summary
    GROUP BY
      id,
    type
  ) i,
  json_each(analysis)
GROUP BY
  id;

-- incident_summary_by_component
DROP VIEW IF EXISTS incident_summary_by_component;
CREATE OR REPLACE VIEW incident_summary_by_component AS
  WITH type_summary AS (
      SELECT summary.id, summary.type, json_object_agg(f.k, f.v) as json
      FROM (
          SELECT evidences.component_id AS id, incidents.type, json_build_object(severity, count(*)) AS severity_agg
          FROM incidents
          INNER JOIN hypotheses ON hypotheses.incident_id = incidents.id
          INNER JOIN evidences ON evidences.hypothesis_id = hypotheses.id
          WHERE (incidents.resolved IS NULL AND incidents.closed IS NULL and evidences.component_id IS NOT NULL
      )
      GROUP BY incidents.severity, incidents.type, evidences.component_id)
      AS summary, json_each(summary.severity_agg) AS f(k,v) GROUP BY summary.type, summary.id
  )

  SELECT id, jsonb_object_agg(key, value) as incidents FROM (select id, json_object_agg(type,json) incidents from type_summary group by id, type) i, json_each(incidents) group by id;

-- Topology view
CREATE OR REPLACE VIEW
  topology AS
WITH
  children AS (
    SELECT
      relationship_id AS id,
      ARRAY_AGG(DISTINCT component_id) AS children
    FROM
      component_relationships
    WHERE
      deleted_at IS NULL
    GROUP BY
      id
  ),
  parents AS (
    SELECT
      component_id AS id,
      ARRAY_AGG(DISTINCT relationship_id) AS parents
    FROM
      component_relationships
    WHERE
      deleted_at IS NULL
    GROUP BY
      id
  ),
  team_info AS (
    SELECT
      team_components.component_id,
      ARRAY_AGG(teams.name) AS team_names
    FROM
      team_components
      LEFT JOIN teams ON team_components.team_id = teams.id
    GROUP BY
      team_components.component_id
),
log_selector_array_elements AS (
  SELECT
    id component_id,
    jsonb_array_elements(
      CASE
        jsonb_typeof(log_selectors)
        WHEN 'array' THEN log_selectors
        ELSE '[]'
      END
    ) AS log_selectors
  FROM
    components
),
log_selectors AS (
  SELECT
    component_id,
    json_agg(jsonb_build_object('name', log_selectors -> 'name')) AS NAMES
  FROM
    log_selector_array_elements
  GROUP BY
    component_id
)
SELECT
  components.*,
  log_selectors.names AS logs,
  checks,
  team_info.team_names,
  incidents,
  analysis,
  children.children,
  parents.parents
FROM
  components
  LEFT JOIN check_summary_by_component ON check_summary_by_component.component_id = components.id
  LEFT JOIN incident_summary_by_component ON incident_summary_by_component.id = components.id
  LEFT JOIN analysis_summary_by_component ON analysis_summary_by_component.id = components.id
  LEFT JOIN children ON children.id = components.id
  LEFT JOIN parents ON parents.id = components.id
  LEFT JOIN team_info ON team_info.component_id = components.id
  LEFT JOIN log_selectors ON log_selectors.component_id = components.id
WHERE
  components.deleted_at IS NULL;
