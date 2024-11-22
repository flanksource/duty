ALTER TABLE config_items DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_changes DISABLE ROW LEVEL SECURITY;

ALTER TABLE config_analysis DISABLE ROW LEVEL SECURITY;

ALTER TABLE components DISABLE ROW LEVEL SECURITY;

-- POLICIES
DROP POLICY IF EXISTS config_items_auth ON config_items;

DROP POLICY IF EXISTS components_auth ON components;

DROP POLICY IF EXISTS config_changes_auth ON config_changes;

DROP POLICY IF EXISTS config_analysis_auth ON config_analysis;

-- View owners
ALTER VIEW analysis_by_config OWNER TO CURRENT_USER;

ALTER VIEW catalog_changes OWNER TO CURRENT_USER;

ALTER VIEW check_summary_by_config OWNER TO CURRENT_USER;

ALTER VIEW check_summary_for_config OWNER TO CURRENT_USER;

ALTER VIEW checks_by_config OWNER TO CURRENT_USER;

ALTER VIEW config_analysis_analyzers OWNER TO CURRENT_USER;

ALTER VIEW config_analysis_by_severity OWNER TO CURRENT_USER;

ALTER VIEW config_analysis_items OWNER TO CURRENT_USER;

ALTER VIEW config_changes_by_types OWNER TO CURRENT_USER;

ALTER VIEW config_changes_items OWNER TO CURRENT_USER;

ALTER VIEW config_class_summary OWNER TO CURRENT_USER;

ALTER VIEW config_classes OWNER TO CURRENT_USER;

ALTER VIEW config_detail OWNER TO CURRENT_USER;

ALTER VIEW config_items_aws OWNER TO CURRENT_USER;

ALTER VIEW config_labels OWNER TO CURRENT_USER;

ALTER VIEW config_names OWNER TO CURRENT_USER;

ALTER VIEW config_scrapers_with_status OWNER TO CURRENT_USER;

ALTER VIEW config_statuses OWNER TO CURRENT_USER;

ALTER VIEW config_summary OWNER TO CURRENT_USER;

ALTER VIEW config_tags OWNER TO CURRENT_USER;

ALTER VIEW config_types OWNER TO CURRENT_USER;

ALTER VIEW configs OWNER TO CURRENT_USER;

ALTER VIEW incidents_by_config OWNER TO CURRENT_USER;

ALTER VIEW incidents_by_config OWNER TO CURRENT_USER;

ALTER VIEW pg_config OWNER TO CURRENT_USER;

---
ALTER MATERIALIZED VIEW config_item_summary_3d OWNER TO CURRENT_USER;

ALTER MATERIALIZED VIEW config_item_summary_7d OWNER TO CURRENT_USER;

ALTER MATERIALIZED VIEW config_item_summary_30d OWNER TO CURRENT_USER;

--
DROP ROLE api_views_owner;

