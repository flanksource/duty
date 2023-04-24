DROP FUNCTION IF EXISTS lookup_component_names;

CREATE
OR REPLACE FUNCTION lookup_component_names (component_id uuid[]) RETURNS TABLE (names text[]) AS $$
BEGIN
    RETURN QUERY
        SELECT array_agg(name) FROM components where id = any( component_id);

END;
$$ language plpgsql;

DROP VIEW IF EXISTS topology;

CREATE OR REPLACE VIEW
  topology AS
WITH
  children AS (
    SELECT
      relationship_id AS id,
      ARRAY_AGG(component_id) AS children
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
      ARRAY_AGG(relationship_id) AS parents
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
  )
SELECT
  components.*,
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
WHERE
  components.deleted_at IS NULL
