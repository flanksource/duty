CREATE OR REPLACE FUNCTION reset_is_pushed_before_update()
RETURNS TRIGGER AS $$
BEGIN
  -- If any column other than is_pushed is changed, reset is_pushed to false.
  IF NEW IS DISTINCT FROM OLD AND NEW.is_pushed IS NOT DISTINCT FROM OLD.is_pushed THEN
    NEW.is_pushed = false;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER reset_is_pushed_before_update
BEFORE UPDATE ON config_analysis
FOR EACH ROW
EXECUTE PROCEDURE reset_is_pushed_before_update();

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
