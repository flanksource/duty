table "evidences" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "description" {
    null = false
    type = text
  }
  column "hypothesis_id" {
    null = false
    type = uuid
  }
  column "config_id" {
    null = true
    type = uuid
  }
  column "config_change_id" {
    null = true
    type = uuid
  }
  column "config_analysis_id" {
    null = true
    type = uuid
  }
  column "component_id" {
    null = true
    type = uuid
  }
  column "check_id" {
    null = true
    type = uuid
  }
  column "definition_of_done" {
    null    = true
    type    = boolean
    default = false
  }
  column "done" {
    null = true
    type = boolean
  }
  column "factor" {
    null = true
    type = boolean
  }
  column "mitigator" {
    null = true
    type = boolean
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "type" {
    null = false
    type = text
  }
  column "evidence" {
    null = true
    type = jsonb
  }
  column "properties" {
    null = true
    type = jsonb
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "script" {
    null = true
    type = text
  }
  column "script_result" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "evidences_check_id_fkey" {
    columns     = [column.check_id]
    ref_columns = [table.checks.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_component_id_fkey" {
    columns     = [column.component_id]
    ref_columns = [table.components.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_config_analysis_id_fkey" {
    columns     = [column.config_analysis_id]
    ref_columns = [table.config_analysis.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_config_change_id_fkey" {
    columns     = [column.config_change_id]
    ref_columns = [table.config_changes.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_config_id_fkey" {
    columns     = [column.config_id]
    ref_columns = [table.config_items.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "evidences_hypothesis_id_fkey" {
    columns     = [column.hypothesis_id]
    ref_columns = [table.hypotheses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "evidences_hypothesis_id_idx" {
    columns = [column.hypothesis_id]
  }
  index "evidences_check_id_idx" {
    columns = [column.check_id]
  }
  index "evidences_component_id_idx" {
    columns = [column.component_id]
  }
  index "evidences_config_analysis_id_idx" {
    columns = [column.config_analysis_id]
  }
  index "evidences_config_change_id_idx" {
    columns = [column.config_change_id]
  }
  index "evidences_config_id_idx" {
    columns = [column.config_id]
  }
  index "evidences_created_by_idx" {
    columns = [column.created_by]
  }
}
table "hypotheses" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "incident_id" {
    null = false
    type = uuid
  }
  column "parent_id" {
    null = true
    type = uuid
  }
  column "owner" {
    null = true
    type = uuid
  }
  column "team_id" {
    null = true
    type = uuid
  }
  column "type" {
    null = false
    type = text
  }
  column "title" {
    null = false
    type = text
  }
  column "status" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "hypotheses_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "hypotheses_incident_id_fkey" {
    columns     = [column.incident_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "hypotheses_owner_fkey" {
    columns     = [column.owner]
    ref_columns = [table.responders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "hypotheses_parent_id_fkey" {
    columns     = [column.parent_id]
    ref_columns = [table.hypotheses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "hypotheses_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  check "hypotheses_type_check" {
    expr = "(type = ANY (ARRAY['root'::text, 'factor'::text, 'solution'::text]))"
  }
  index "hypotheses_incident_id_idx" {
    columns = [column.incident_id]
  }
  index "hypotheses_parent_id_idx" {
    columns = [column.parent_id]
  }
  index "hypotheses_created_by_idx" {
    columns = [column.created_by]
  }
  index "hypotheses_owner_idx" {
    columns = [column.owner]
  }
  index "hypotheses_team_id_idx" {
    columns = [column.team_id]
  }
}

table "incident_histories" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "incident_id" {
    null = false
    type = uuid
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "type" {
    null = true
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "hypothesis_id" {
    null = true
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "responder_id" {
    null = true
    type = uuid
  }
  column "evidence_id" {
    null = true
    type = uuid
  }
  column "comment_id" {
    null = true
    type = uuid
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "incident_histories_comment_id_fkey" {
    columns     = [column.comment_id]
    ref_columns = [table.comments.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_histories_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_histories_evidence_id_fkey" {
    columns     = [column.evidence_id]
    ref_columns = [table.evidences.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_histories_hypothesis_id_fkey" {
    columns     = [column.hypothesis_id]
    ref_columns = [table.hypotheses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_histories_incident_id_fkey" {
    columns     = [column.incident_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_histories_responder_id_fkey" {
    columns     = [column.responder_id]
    ref_columns = [table.responders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "incident_histories_incident_id_idx" {
    columns = [column.incident_id]
  }
  index "incident_histories_comment_id_idx" {
    columns = [column.comment_id]
  }
  index "incident_histories_created_by_idx" {
    columns = [column.created_by]
  }
  index "incident_histories_evidence_id_idx" {
    columns = [column.evidence_id]
  }
  index "incident_histories_hypothesis_id_idx" {
    columns = [column.hypothesis_id]
  }
  index "incident_histories_responder_id_idx" {
    columns = [column.responder_id]
  }
}

table "incident_relationships" {
  schema = schema.public
  column "incident_id" {
    null = false
    type = uuid
  }
  column "related_id" {
    null = false
    type = uuid
  }
  column "relationship" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  foreign_key "incident_relationships_incident_id_fkey" {
    columns     = [column.incident_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incident_relationships_related_id_fkey" {
    columns     = [column.related_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "incident_relationships_incident_id_idx" {
    columns = [column.incident_id]
  }
  index "incident_relationships_related_id_idx" {
    columns = [column.related_id]
  }
}
table "incident_rules" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "name" {
    null = false
    type = text
  }
  column "spec" {
    null = true
    type = jsonb
  }
  column "source" {
    null = true
    type = text
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "incident_rules_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "incident_rules_name_key" {
    unique  = true
    columns = [column.name]
  }
  index "incident_rules_created_by_idx" {
    columns = [column.created_by]
  }
}

table "incidents" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "incident_id" {
    null    = false
    type    = varchar(10)
    default = sql("format_incident_id(NEXTVAL('incident_id_sequence'))")
  }
  column "incident_rule_id" {
    null = true
    type = uuid
  }
  column "title" {
    null = false
    type = text
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "commander_id" {
    null = true
    type = uuid
  }
  column "communicator_id" {
    null = true
    type = uuid
  }
  column "severity" {
    null = false
    type = text
  }
  column "description" {
    null = false
    type = text
  }
  column "type" {
    null = false
    type = text
  }
  column "status" {
    null = false
    type = text
  }
  column "acknowledged" {
    null = true
    type = timestamptz
  }
  column "resolved" {
    null = true
    type = timestamptz
  }
  column "closed" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  index "incidents_incident_id_key" {
    unique  = true
    columns = [column.incident_id]
  }
  index "incidents_commander_id_idx" {
    columns = [column.commander_id]
  }
  index "incidents_communicator_id_idx" {
    columns = [column.communicator_id]
  }
  index "incidents_created_by_idx" {
    columns = [column.created_by]
  }
  index "incidents_incident_rule_id_idx" {
    columns = [column.incident_rule_id]
  }
  foreign_key "incidents_commander_id_fkey" {
    columns     = [column.commander_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incidents_communicator_id_fkey" {
    columns     = [column.communicator_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incidents_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "incidents_incident_rule_id_fkey" {
    columns     = [column.incident_rule_id]
    ref_columns = [table.incident_rules.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "responders" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "incident_id" {
    null = false
    type = uuid
  }
  column "type" {
    null = false
    type = text
  }
  column "index" {
    null = true
    type = smallint
  }
  column "person_id" {
    null = true
    type = uuid
  }
  column "team_id" {
    null = true
    type = uuid
  }
  column "external_id" {
    null = true
    type = text
  }
  column "properties" {
    null = true
    type = jsonb
  }
  column "acknowledged" {
    null = true
    type = timestamptz
  }
  column "resolved" {
    null = true
    type = timestamptz
  }
  column "closed" {
    null = true
    type = timestamptz
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "deleted_at" {
    null = true
    type = timestamptz
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "responders_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "responders_incident_id_fkey" {
    columns     = [column.incident_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "responders_person_id_fkey" {
    columns     = [column.person_id]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "responders_team_id_fkey" {
    columns     = [column.team_id]
    ref_columns = [table.teams.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "responders_incident_id_idx" {
    columns = [column.incident_id]
  }
  index "responders_created_by_idx" {
    columns = [column.created_by]
  }
  index "responders_person_id_idx" {
    columns = [column.person_id]
  }
  index "responders_team_id_idx" {
    columns = [column.team_id]
  }
}

table "comment_responders" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "comment_id" {
    null = false
    type = uuid
  }
  column "responder_id" {
    null = false
    type = uuid
  }
  column "external_id" {
    null = true
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "comment_responders_comment_id_fkey" {
    columns     = [column.comment_id]
    ref_columns = [table.comments.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "comment_responders_responder_id_fkey" {
    columns     = [column.responder_id]
    ref_columns = [table.responders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "comment_responders_comment_id_idx" {
    columns = [column.comment_id]
  }
  index "comment_responders_responder_id_idx" {
    columns = [column.responder_id]
  }
}
table "comments" {
  schema = schema.public
  column "id" {
    null    = false
    type    = uuid
    default = sql("generate_ulid()")
  }
  column "created_by" {
    null = false
    type = uuid
  }
  column "comment" {
    null = false
    type = text
  }
  column "external_id" {
    null = true
    type = text
  }
  column "external_created_by" {
    null = true
    type = text
  }
  column "incident_id" {
    null = false
    type = uuid
  }
  column "responder_id" {
    null = true
    type = uuid
  }
  column "hypothesis_id" {
    null = true
    type = uuid
  }
  column "read" {
    null = true
    type = sql("smallint[]")
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "comments_created_by_fkey" {
    columns     = [column.created_by]
    ref_columns = [table.people.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "comments_hypothesis_id_fkey" {
    columns     = [column.hypothesis_id]
    ref_columns = [table.hypotheses.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "comments_incident_id_fkey" {
    columns     = [column.incident_id]
    ref_columns = [table.incidents.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  foreign_key "comments_responder_id_fkey" {
    columns     = [column.responder_id]
    ref_columns = [table.responders.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
  index "comments_incident_id_idx" {
    columns = [column.incident_id]
  }
  index "comments_hypothesis_id_idx" {
    columns = [column.hypothesis_id]
  }
  index "comments_created_by_idx" {
    columns = [column.created_by]
  }
  index "comments_responder_id_idx" {
    columns = [column.responder_id]
  }
}

table "severities" {
  schema = schema.public
  column "id" {
    null = true
    type = integer
  }
  column "name" {
    null = false
    type = text
  }
  column "aliases" {
    null = true
    type = sql("text[]")
  }
  column "icon" {
    null = true
    type = text
  }
}
