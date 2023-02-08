package duty

import "fmt"

const (
	emptyJSONBArraySQL  = "(SELECT json_build_array())::jsonb"
	emptyJSONBObjectSQL = "(SELECT json_build_object())::jsonb"
)

type TopologyOptions struct {
	ID                     string            `query:"id"`
	Owner                  string            `query:"owner"`
	Labels                 map[string]string `query:"labels"`
	Status                 []string          `query:"status"`
	Types                  []string          `query:"types"`
	Depth                  int               `query:"depth"`
	Flatten                bool              `query:"flatten"`
	IncludeHealth          bool              `query:"includeHealth"`
	IncludeInsightsSummary bool              `query:"includeInsightsSummary"`
}

func (opt TopologyOptions) String() string {
	return fmt.Sprintf("%#v", opt)
}

func (opt TopologyOptions) componentWhereClause() string {
	s := "where components.deleted_at is null "
	if opt.ID != "" {
		s += `and (starts_with(path,
			(SELECT
				(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
				FROM components where id = :id)
			) or id = :id or path = :id :: text)`
	}
	if opt.Owner != "" {
		s += " AND (components.owner = :owner or id = :id)"
	}
	if opt.Labels != nil {
		s += " AND (components.labels @> :labels"
		if opt.ID != "" {
			s += " or id = :id"
		}
		s += ")"
	}
	return s
}

func (opt TopologyOptions) componentRelationWhereClause() string {
	s := "WHERE component_relationships.deleted_at IS NULL"
	if opt.Owner != "" {
		s += " AND (parent.owner = :owner)"
	}
	if opt.Labels != nil {
		s += " AND (parent.labels @> :labels)"
	}
	if opt.ID != "" {
		s += ` and (component_relationships.relationship_id = :id or starts_with(component_relationships.relationship_path, (SELECT
			(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
			FROM components where id = :id)))`
	} else {
		s += ` and (parent.parent_id is null or starts_with(component_relationships.relationship_path, (SELECT
			(CASE WHEN (path IS NULL OR path = '') THEN id :: text ELSE concat(path,'.', id) END)
			FROM components where id = parent.id)))`
	}
	return s
}

func TopologyQuery(opts TopologyOptions) (string, map[string]any) {
	query := fmt.Sprintf(`
    WITH topology_result as (
    SELECT * FROM components %s
	UNION (
        SELECT components.* FROM component_relationships
        INNER JOIN components ON components.id = component_relationships.component_id
        INNER JOIN components AS parent ON component_relationships.relationship_id = parent.id %s)
    )
	SELECT json_agg(
        jsonb_set_lax(
            jsonb_set_lax(
                jsonb_set_lax(
                    to_jsonb(topology_result),
                        '{checks}', %s
                ), '{summary,insights}', %s
            ), '{summary,incidents}', %s
        )
    ) :: jsonb FROM topology_result`,
		opts.componentWhereClause(), opts.componentRelationWhereClause(), opts.checksForComponents(),
		opts.configAnalysisSummaryForComponents(), opts.incidentSummaryForComponents())

	args := make(map[string]any)
	if opts.ID != "" {
		args["id"] = opts.ID
	}
	if opts.Owner != "" {
		args["owner"] = opts.Owner
	}
	if opts.Labels != nil {
		args["labels"] = opts.Labels
	}
	return query, args
}

func (opts TopologyOptions) checksForComponents() string {
	return `(
        SELECT json_agg(checks) FROM checks
        LEFT JOIN check_component_relationships ON checks.id = check_component_relationships.check_id
        WHERE check_component_relationships.component_id = topology_result.id AND check_component_relationships.deleted_at IS NULL
        GROUP BY check_component_relationships.component_id
    ) :: jsonb`
}

func (opts TopologyOptions) configAnalysisSummaryForComponents() string {
	if !opts.IncludeInsightsSummary {
		return emptyJSONBObjectSQL
	}
	return `(
        SELECT json_object_agg(flatten.analysis_type, flatten.summary_json)
        FROM (
            SELECT summary.component_id, summary.analysis_type, json_object_agg(f.k, f.v) as summary_json
            FROM (
                SELECT config_component_relationships.component_id AS component_id, config_analysis.analysis_type, json_build_object(severity, count(*)) AS severity_agg
                FROM config_analysis
                LEFT JOIN config_component_relationships ON config_analysis.config_id = config_component_relationships.config_id
                WHERE config_component_relationships.component_id = topology_result.id AND config_component_relationships.deleted_at IS NULL
                GROUP BY config_analysis.severity, config_analysis.analysis_type, config_component_relationships.component_id
            ) AS summary, json_each(summary.severity_agg) AS f(k,v) GROUP BY summary.analysis_type, summary.component_id
        ) AS flatten GROUP BY flatten.component_id
    ) :: jsonb`
}

func (p TopologyOptions) incidentSummaryForComponents() string {
	return `(
        SELECT json_object_agg(flatten.type, flatten.summary_json)
        FROM (
            SELECT summary.component_id, summary.type, json_object_agg(f.k, f.v) as summary_json
            FROM (
                SELECT evidences.component_id AS component_id, incidents.type, json_build_object(severity, count(*)) AS severity_agg
                FROM incidents
                INNER JOIN hypotheses ON hypotheses.incident_id = incidents.id
                INNER JOIN evidences ON evidences.hypothesis_id = hypotheses.id
                WHERE evidences.component_id = topology_result.id AND (incidents.resolved IS NULL AND incidents.closed IS NULL)
                GROUP BY incidents.severity, incidents.type, evidences.component_id
            ) AS summary, json_each(summary.severity_agg) AS f(k,v) GROUP BY summary.type, summary.component_id
        ) AS flatten GROUP BY flatten.component_id
    ) :: jsonb`
}
