DROP FUNCTION if exists lookup_component_incidents;

CREATE OR REPLACE FUNCTION lookup_component_incidents(component_id text)
RETURNS TABLE (
    id UUID
) AS $$
BEGIN
    RETURN QUERY
        SELECT incidents.id FROM incidents WHERE incidents.id IN (
            SELECT incident_id FROM hypotheses WHERE hypotheses.id IN (
                SELECT hypothesis_id FROM evidences WHERE evidences.component_id = $1::UUID
            )
        );
END;
$$
language plpgsql;


-- incidents_by_component
DROP VIEW IF EXISTS incidents_by_component;
CREATE OR REPLACE VIEW incidents_by_component AS
  SELECT DISTINCT incidents.title,incidents.id, incidents.created_at, incidents."type", incidents.status, incidents.severity, component_id FROM evidences
  INNER join hypotheses on evidences.hypothesis_id = hypotheses.id
  INNER JOIN incidents on hypotheses.incident_id = incidents.id
  WHERE component_id is not null;


--incidents_by_config
DROP VIEW IF EXISTS incidents_by_config;
CREATE OR REPLACE VIEW incidents_by_config AS
  SELECT DISTINCT incidents.title, incidents.id, incidents.created_at, incidents."type", incidents.status,  incidents.severity, config_id FROM evidences
  INNER join hypotheses on evidences.hypothesis_id = hypotheses.id
  INNER JOIN incidents on hypotheses.incident_id = incidents.id
  WHERE evidences.config_id is not null;

-- incident_summary VIEW
DROP VIEW IF EXISTS incident_summary;
CREATE OR REPLACE VIEW incident_summary AS 
  WITH distinct_commenter AS (
    SELECT
      DISTINCT ON (people.id) people.id,
      people.avatar,
      people.name,
      comments.incident_id
    FROM
      comments
      LEFT JOIN people ON comments.created_by = people.id
    WHERE
      people.id IS NOT NULL
    ORDER BY
      people.id
  ),
  commenters AS (
    SELECT
      incident_id,
      jsonb_agg(
        jsonb_build_object(
          'id', distinct_commenter.id,
          'avatar', distinct_commenter.avatar,
          'name', distinct_commenter.name
        ) ORDER BY distinct_commenter.id
      ) AS actor
    FROM
      distinct_commenter
    GROUP BY
      incident_id
  ),
  distinct_responder AS (
    SELECT
      DISTINCT ON (people.id, responders.incident_id) people.id as id,
      people.avatar,
      people.name,
      responders.incident_id
    FROM
      responders
      INNER JOIN people ON responders.person_id = people.id
    UNION
    SELECT
      DISTINCT ON (teams.id, responders.incident_id) teams.id as id,
      teams.icon as avatar,
      teams.name,
      responders.incident_id
    FROM
      responders
      INNER JOIN teams ON responders.team_id = teams.id
  ),
  responders AS (
    SELECT
      incident_id,
      jsonb_agg(
        jsonb_build_object(
          'id', distinct_responder.id,
          'avatar', distinct_responder.avatar,
          'name', distinct_responder.name
        ) ORDER BY distinct_responder.id
      ) AS actor
    FROM
      distinct_responder
    GROUP BY
      incident_id
  )
  SELECT
    incidents.id,
    incidents.incident_id,
    incidents.title,
    incidents.severity,
    incidents.type,
    incidents.status,
    incidents.created_at,
    incidents.updated_at,
    jsonb_build_object(
      'id', people.id,
      'avatar', people.avatar,
      'name', people.name
    ) AS commander,
    responders.actor responders,
    commenters.actor commenters
  FROM
    incidents
    LEFT JOIN people ON incidents.commander_id = people.id
    LEFT JOIN responders ON incidents.id = responders.incident_id
    LEFT JOIN commenters ON commenters.incident_id = incidents.id
  GROUP BY
    incidents.id,
    people.id,
    commenters.actor,
    responders.actor;
