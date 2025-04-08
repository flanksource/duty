package rbac

import (
	"net/http"

	"github.com/flanksource/duty/rbac/policy"
)

var dbResourceObjMap = map[string]string{
	"access_token":                      policy.ObjectAuthConfidential,
	"access_tokens":                     policy.ObjectAuthConfidential,
	"agents_summary":                    policy.ObjectMonitor,
	"agents":                            policy.ObjectDatabasePublic,
	"analysis_by_component":             policy.ObjectCatalog,
	"analysis_by_config":                policy.ObjectCatalog,
	"analysis_summary_by_component":     policy.ObjectCatalog,
	"analysis_types":                    policy.ObjectDatabasePublic,
	"analyzer_types":                    policy.ObjectDatabasePublic,
	"artifacts":                         policy.ObjectArtifact,
	"canaries_with_status":              policy.ObjectCanary,
	"canaries":                          policy.ObjectCanary,
	"casbin_rule":                       policy.ObjectAuth,
	"catalog_changes":                   policy.ObjectCatalog,
	"change_types":                      policy.ObjectDatabasePublic,
	"changes_by_component":              policy.ObjectCatalog,
	"check_component_relationships":     policy.ObjectCanary,
	"check_config_relationships":        policy.ObjectCanary,
	"check_labels":                      policy.ObjectDatabasePublic,
	"check_names":                       policy.ObjectDatabasePublic,
	"check_status_summary_hour":         policy.ObjectCanary,
	"check_statuses_1d":                 policy.ObjectCanary,
	"check_statuses_1h":                 policy.ObjectCanary,
	"check_statuses_5m":                 policy.ObjectCanary,
	"check_statuses":                    policy.ObjectCanary,
	"check_summary_by_component":        policy.ObjectCanary,
	"check_summary_by_config":           policy.ObjectCatalog,
	"check_summary_for_config":          policy.ObjectCatalog,
	"check_summary":                     policy.ObjectCanary,
	"checks_by_component":               policy.ObjectCanary,
	"checks_by_config":                  policy.ObjectCanary,
	"checks_status_artifacts":           policy.ObjectCanary,
	"checks":                            policy.ObjectCanary,
	"comment_responders":                policy.ObjectIncident,
	"comments":                          policy.ObjectIncident,
	"component_labels":                  policy.ObjectDatabasePublic,
	"component_names_all":               policy.ObjectTopology,
	"component_names":                   policy.ObjectDatabasePublic,
	"component_relationships":           policy.ObjectTopology,
	"component_types":                   policy.ObjectDatabasePublic,
	"components_with_logs":              policy.ObjectTopology,
	"components":                        policy.ObjectTopology,
	"config_analysis_analyzers":         policy.ObjectCatalog,
	"config_analysis_by_severity":       policy.ObjectCatalog,
	"config_analysis_items":             policy.ObjectCatalog,
	"config_analysis":                   policy.ObjectCatalog,
	"config_changes_by_types":           policy.ObjectCatalog,
	"config_changes_items":              policy.ObjectCatalog,
	"config_changes":                    policy.ObjectCatalog,
	"config_class_summary":              policy.ObjectCatalog,
	"config_classes":                    policy.ObjectDatabasePublic,
	"config_component_relationships":    policy.ObjectCatalog,
	"config_detail":                     policy.ObjectCatalog,
	"config_items_aws":                  policy.ObjectCatalog,
	"config_items":                      policy.ObjectCatalog,
	"config_labels":                     policy.ObjectDatabasePublic,
	"config_names":                      policy.ObjectDatabasePublic,
	"config_relationships":              policy.ObjectCatalog,
	"config_scrapers_with_status":       policy.ObjectMonitor,
	"config_scrapers":                   policy.ObjectDatabaseSettings,
	"config_statuses":                   policy.ObjectDatabasePublic,
	"config_summary":                    policy.ObjectCatalog,
	"config_tags":                       policy.ObjectDatabasePublic,
	"config_tags_labels_keys":           policy.ObjectDatabasePublic,
	"component_labels_keys":             policy.ObjectDatabasePublic,
	"checks_labels_keys":                policy.ObjectDatabasePublic,
	"config_types":                      policy.ObjectDatabasePublic,
	"configs":                           policy.ObjectCatalog,
	"connections_list":                  policy.ObjectDatabasePublic,
	"connections":                       policy.ObjectConnection,
	"connection_details":                policy.ObjectConnectionDetail,
	"courier_message_dispatches":        policy.ObjectAuthConfidential,
	"courier_messaged_dispatches":       policy.ObjectAuthConfidential,
	"courier_messages":                  policy.ObjectAuthConfidential,
	"event_queue_summary":               policy.ObjectMonitor,
	"event_queue":                       policy.ObjectDatabaseSystem,
	"evidences":                         policy.ObjectIncident,
	"failed_events":                     policy.ObjectMonitor,
	"hypotheses":                        policy.ObjectIncident,
	"identities":                        policy.ObjectDatabasePublic,
	"identity_credential_identifiers":   policy.ObjectAuthConfidential,
	"identity_credential_types":         policy.ObjectAuthConfidential,
	"identity_credentials":              policy.ObjectAuthConfidential,
	"identity_recovery_addresses":       policy.ObjectAuthConfidential,
	"identity_recovery_codes":           policy.ObjectAuthConfidential,
	"identity_recovery_tokens":          policy.ObjectAuthConfidential,
	"identity_verifiable_addresses":     policy.ObjectAuthConfidential,
	"identity_verification_codes":       policy.ObjectAuthConfidential,
	"identity_verification_tokens":      policy.ObjectAuthConfidential,
	"incident_histories":                policy.ObjectIncident,
	"incident_relationships":            policy.ObjectIncident,
	"incident_rules":                    policy.ObjectIncident,
	"incident_summary_by_component":     policy.ObjectIncident,
	"incident_summary":                  policy.ObjectIncident,
	"incidents_by_component":            policy.ObjectIncident,
	"incidents_by_config":               policy.ObjectIncident,
	"incidents":                         policy.ObjectIncident,
	"integrations_with_status":          policy.ObjectMonitor,
	"integrations":                      policy.ObjectMonitor,
	"job_histories":                     policy.ObjectMonitor,
	"job_history_latest_status":         policy.ObjectMonitor,
	"job_history_names":                 policy.ObjectMonitor,
	"job_history":                       policy.ObjectMonitor,
	"logging_backends":                  policy.ObjectDatabaseSettings,
	"migration_logs":                    policy.ObjectDatabaseSystem,
	"networks":                          policy.ObjectAuthConfidential,
	"notification_send_history":         policy.ObjectMonitor,
	"notification_send_history_summary": policy.ObjectMonitor,
	"notifications_summary":             policy.ObjectMonitor,
	"notifications":                     policy.ObjectNotification,
	"notification_groups":               policy.ObjectNotification,
	"notification_group_resources":      policy.ObjectNotification,
	"notification_silences":             policy.ObjectNotification,
	"people_roles":                      policy.ObjectDatabasePublic,
	"people":                            policy.ObjectPeople,
	"permissions":                       policy.ObjectDatabaseSystem,
	"permission_groups":                 policy.ObjectDatabaseSystem,
	"permissions_summary":               policy.ObjectDatabaseSystem,
	"permissions_group_summary":         policy.ObjectDatabaseSystem,
	"playbook_action_agent_data":        policy.ObjectPlaybooks,
	"playbook_approvals":                policy.ObjectPlaybooks,
	"playbook_names":                    policy.ObjectDatabasePublic,
	"playbook_run_actions":              policy.ObjectPlaybooks,
	"playbook_runs":                     policy.ObjectPlaybooks,
	"playbooks_for_agent":               policy.ObjectAgentPush,
	"rpc/get_playbook_run_actions":      policy.ObjectPlaybooks,
	"playbooks":                         policy.ObjectPlaybooks,
	"properties":                        policy.ObjectDatabaseSystem,
	"push_queue_summary":                policy.ObjectMonitor,
	"responders":                        policy.ObjectIncident,
	"rpc/lookup_component_config_id_related_components": policy.ObjectTopology,
	"rpc/_related_config_ids_recursive":                 policy.ObjectCatalog,
	"rpc/check_summary_for_component":                   policy.ObjectCanary,
	"rpc/config_relationships_recursive":                policy.ObjectCatalog,
	"rpc/get_recursive_path":                            policy.ObjectCatalog,
	"rpc/lookup_analysis_by_component":                  policy.ObjectTopology,
	"rpc/lookup_changes_by_component":                   policy.ObjectTopology,
	"rpc/lookup_component_by_property":                  policy.ObjectTopology,
	"rpc/lookup_component_children":                     policy.ObjectTopology,
	"rpc/lookup_component_incidents":                    policy.ObjectTopology,
	"rpc/lookup_component_names":                        policy.ObjectTopology,
	"rpc/lookup_component_relations":                    policy.ObjectTopology,
	"rpc/lookup_components_by_check":                    policy.ObjectTopology,
	"rpc/lookup_components_by_config":                   policy.ObjectTopology,
	"rpc/lookup_config_children":                        policy.ObjectCatalog,
	"rpc/lookup_config_relations":                       policy.ObjectCatalog,
	"rpc/lookup_configs_by_component":                   policy.ObjectTopology,
	"rpc/lookup_related_configs":                        policy.ObjectCatalog,
	"rpc/related_changes_recursive":                     policy.ObjectCatalog,
	"rpc/related_config_ids_recursive":                  policy.ObjectCatalog,
	"rpc/related_config_ids":                            policy.ObjectCatalog,
	"rpc/related_configs_recursive":                     policy.ObjectCatalog,
	"rpc/related_configs":                               policy.ObjectCatalog,
	"rpc/soft_delete_canary":                            policy.ObjectCanary,
	"rpc/soft_delete_check":                             policy.ObjectCanary,
	"rpc/uuid_to_ulid":                                  policy.ObjectDatabasePublic,
	"saved_query":                                       policy.ObjectDatabasePublic,
	"schema_migration":                                  policy.ObjectAuthConfidential,
	"scrape_plugins":                                    policy.ObjectCatalog,
	"selfservice_errors":                                policy.ObjectAuthConfidential,
	"selfservice_login_flows":                           policy.ObjectAuthConfidential,
	"selfservice_recovery_flows":                        policy.ObjectAuthConfidential,
	"selfservice_registration_flows":                    policy.ObjectAuthConfidential,
	"selfservice_settings_flows":                        policy.ObjectAuthConfidential,
	"selfservice_verification_flows":                    policy.ObjectAuthConfidential,
	"session_devices":                                   policy.ObjectAuthConfidential,
	"sessions":                                          policy.ObjectAuthConfidential,
	"severities":                                        policy.ObjectDatabasePublic,
	"team_components":                                   policy.ObjectDatabasePublic,
	"team_members":                                      policy.ObjectDatabasePublic,
	"teams_with_status":                                 policy.ObjectDatabasePublic,
	"teams":                                             policy.ObjectDatabasePublic,
	"topologies_with_status":                            policy.ObjectTopology,
	"topologies":                                        policy.ObjectTopology,
	"topology":                                          policy.ObjectTopology,
}

func GetObjectByTable(resource string) string {
	if v, exists := dbResourceObjMap[resource]; exists {
		return v
	}
	return ""
}

func GetActionFromHttpMethod(method string) string {
	switch method {
	case http.MethodGet:
		return policy.ActionRead
	case http.MethodPatch:
		return policy.ActionUpdate
	case http.MethodPost:
		return policy.ActionCreate
	case http.MethodDelete:
		return policy.ActionDelete
	}

	return ""
}
