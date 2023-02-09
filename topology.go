package duty

import "fmt"

type TopologyOptions struct {
	ID     string            `query:"id"`
	Owner  string            `query:"owner"`
	Labels map[string]string `query:"labels"`
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
            INNER JOIN components AS parent ON component_relationships.relationship_id = parent.id %s
        )
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
	return `(SELECT analysis FROM analysis_summary_by_component WHERE id = topology_result.id)`
}

func (p TopologyOptions) incidentSummaryForComponents() string {
	return `(SELECT incidents FROM incident_summary_by_component WHERE id = topology_result.id)`
}
