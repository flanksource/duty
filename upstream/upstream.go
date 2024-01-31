package upstream

import (
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/http"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

type UpstreamConfig struct {
	AgentName          string
	Host               string
	InsecureSkipVerify bool
	Username           string
	Password           string
	Labels             []string
	Debug              bool
	Options            []func(c *http.Client)
}

func (t UpstreamConfig) String() string {
	var s []string

	if t.Host != "" {
		s = append(s, fmt.Sprintf("host=%s", t.Host))
	}
	if t.Username != "" {
		s = append(s, fmt.Sprintf("user=%s", t.Username))
	}

	if t.Password != "" {
		s = append(s, fmt.Sprintf("pass=%s***", t.Password[0:1]))
	}

	if t.AgentName != "" {
		s = append(s, fmt.Sprintf("agent=%s", t.AgentName))
	}
	if len(t.Labels) > 0 {
		s = append(s, fmt.Sprintf("labels=%v", t.Labels))
	}
	return strings.Join(s, " ")
}

func (t *UpstreamConfig) Valid() bool {
	return t.Host != "" && t.Username != "" && t.Password != "" && t.AgentName != ""
}

func (t *UpstreamConfig) IsPartiallyFilled() bool {
	return !t.Valid() && (t.Host != "" || t.Password != "" || t.AgentName != "")
}

func (t *UpstreamConfig) LabelsMap() map[string]string {
	return collections.KeyValueSliceToMap(t.Labels)
}

// PushData consists of data about changes to
// components, configs, analysis.
type PushData struct {
	AgentName                    string                               `json:"agent_name,omitempty"`
	Canaries                     []models.Canary                      `json:"canaries,omitempty"`
	Checks                       []models.Check                       `json:"checks,omitempty"`
	Components                   []models.Component                   `json:"components,omitempty"`
	ConfigScrapers               []models.ConfigScraper               `json:"config_scrapers,omitempty"`
	ConfigAnalysis               []models.ConfigAnalysis              `json:"config_analysis,omitempty"`
	ConfigChanges                []models.ConfigChange                `json:"config_changes,omitempty"`
	ConfigItems                  []models.ConfigItem                  `json:"config_items,omitempty"`
	CheckStatuses                []models.CheckStatus                 `json:"check_statuses,omitempty"`
	ConfigRelationships          []models.ConfigRelationship          `json:"config_relationships,omitempty"`
	ComponentRelationships       []models.ComponentRelationship       `json:"component_relationships,omitempty"`
	ConfigComponentRelationships []models.ConfigComponentRelationship `json:"config_component_relationships,omitempty"`
	Topologies                   []models.Topology                    `json:"topologies,omitempty"`
	PlaybookActions              []models.PlaybookRunAction           `json:"playbook_actions,omitempty"`
	Artifacts                    []models.Artifact                    `json:"artifacts,omitempty"`
}

func (p *PushData) AddMetrics(counter context.Counter) {
	counter.Label("table", "artifacts").Add(len(p.Artifacts))
	counter.Label("table", "canaries").Add(len(p.Canaries))
	counter.Label("table", "check_statuses").Add(len(p.CheckStatuses))
	counter.Label("table", "checks").Add(len(p.Checks))
	counter.Label("table", "component_relationships").Add(len(p.ComponentRelationships))
	counter.Label("table", "components").Add(len(p.Components))
	counter.Label("table", "config_analysis").Add(len(p.ConfigAnalysis))
	counter.Label("table", "config_changes").Add(len(p.ConfigChanges))
	counter.Label("table", "config_component_relationships").Add(len(p.ConfigComponentRelationships))
	counter.Label("table", "config_items").Add(len(p.ConfigItems))
	counter.Label("table", "config_relationships").Add(len(p.ConfigRelationships))
	counter.Label("table", "config_scrapers").Add(len(p.ConfigScrapers))
	counter.Label("table", "playbook_actions").Add(len(p.PlaybookActions))
	counter.Label("table", "topologies").Add(len(p.Topologies))
}
func (p *PushData) String() string {
	result := ""
	for k, v := range p.Attributes() {
		result += fmt.Sprintf("%s=%v ", k, v)
	}
	return strings.TrimSpace(result)
}

func (p *PushData) Attributes() map[string]any {
	attrs := map[string]any{
		"name": p.AgentName,
	}

	if len(p.Topologies) > 0 {
		attrs["Topologies"] = len(p.Topologies)
	}
	if len(p.Canaries) > 0 {
		attrs["Canaries"] = len(p.Canaries)
	}
	if len(p.Checks) > 0 {
		attrs["Checks"] = len(p.Checks)
	}
	if len(p.Components) > 0 {
		attrs["Components"] = len(p.Components)
	}
	if len(p.ConfigAnalysis) > 0 {
		attrs["ConfigAnalysis"] = len(p.ConfigAnalysis)
	}
	if len(p.ConfigScrapers) > 0 {
		attrs["ConfigScrapers"] = len(p.ConfigScrapers)
	}
	if len(p.ConfigChanges) > 0 {
		attrs["ConfigChanges"] = len(p.ConfigChanges)
	}
	if len(p.ConfigItems) > 0 {
		attrs["ConfigItems"] = len(p.ConfigItems)
	}
	if len(p.CheckStatuses) > 0 {
		attrs["CheckStatuses"] = len(p.CheckStatuses)
	}
	if len(p.ConfigRelationships) > 0 {
		attrs["ConfigRelationships"] = len(p.ConfigRelationships)
	}
	if len(p.ComponentRelationships) > 0 {
		attrs["ComponentRelationships"] = len(p.ComponentRelationships)
	}
	if len(p.ConfigComponentRelationships) > 0 {
		attrs["ConfigComponentRelationships"] = len(p.ConfigComponentRelationships)
	}
	if len(p.Artifacts) > 0 {
		attrs["Artifacts"] = len(p.Artifacts)
	}

	return attrs
}

func (t *PushData) Count() int {
	return len(t.Canaries) + len(t.Checks) + len(t.Components) + len(t.ConfigScrapers) +
		len(t.ConfigAnalysis) + len(t.ConfigChanges) + len(t.ConfigItems) + len(t.CheckStatuses) +
		len(t.ConfigRelationships) + len(t.ComponentRelationships) + len(t.ConfigComponentRelationships) +
		len(t.Topologies) + len(t.PlaybookActions) + len(t.Artifacts)
}

// ReplaceTopologyID replaces the topology_id for all the components
// with the provided id.
func (t *PushData) ReplaceTopologyID(id *uuid.UUID) {
	for i := range t.Components {
		t.Components[i].TopologyID = id
	}
}

// PopulateAgentID sets agent_id on all the data
func (t *PushData) PopulateAgentID(id uuid.UUID) {
	for i := range t.Canaries {
		t.Canaries[i].AgentID = id
	}
	for i := range t.Checks {
		t.Checks[i].AgentID = id
	}
	for i := range t.Components {
		t.Components[i].AgentID = id
	}
	for i := range t.ConfigItems {
		t.ConfigItems[i].AgentID = id
	}
	for i := range t.ConfigScrapers {
		t.ConfigScrapers[i].AgentID = id
	}
	for i := range t.Topologies {
		t.Topologies[i].AgentID = id
	}
}

// ApplyLabels injects additional labels to the suitable fields
func (t *PushData) ApplyLabels(labels map[string]string) {
	for i := range t.Components {
		t.Components[i].Labels = collections.MergeMap(t.Components[i].Labels, labels)
	}

	for i := range t.Checks {
		t.Checks[i].Labels = collections.MergeMap(t.Checks[i].Labels, labels)
	}

	for i := range t.Canaries {
		t.Canaries[i].Labels = collections.MergeMap(t.Canaries[i].Labels, labels)
	}

	for i := range t.Topologies {
		t.Topologies[i].Labels = collections.MergeMap(t.Topologies[i].Labels, labels)
	}
}

func GetPrimaryKeysHash(ctx context.Context, req PaginateRequest, agentID uuid.UUID) (*PaginateResponse, error) {
	query := fmt.Sprintf(`
		WITH p_keys AS (
 			SELECT id::TEXT, COALESCE(updated_at::text, '') as updated_at
			FROM %s
			WHERE id::TEXT > ? AND agent_id = ?
			ORDER BY id
			LIMIT ?
		)
		SELECT
			encode(digest(string_agg(id || updated_at, ''), 'sha256'), 'hex') as sha256sum,
			MAX(id) as last_id,
			COUNT(*) as total
		FROM
			p_keys`, req.Table)

	var resp PaginateResponse
	err := ctx.DB().Raw(query, req.From, agentID, req.Size).Scan(&resp).Error
	return &resp, err
}

func GetMissingResourceIDs(ctx context.Context, ids []string, paginateReq PaginateRequest) (*PushData, error) {
	var pushData PushData

	tx := ctx.DB().Where("agent_id = ?", uuid.Nil)
	switch paginateReq.Table {
	case "topologies":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.Topologies).Error; err != nil {
			return nil, fmt.Errorf("error fetching topologies: %w", err)
		}

	case "canaries":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.Canaries).Error; err != nil {
			return nil, fmt.Errorf("error fetching canaries: %w", err)
		}

	case "checks":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.Checks).Error; err != nil {
			return nil, fmt.Errorf("error fetching checks: %w", err)
		}

	case "components":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.Components).Error; err != nil {
			return nil, fmt.Errorf("error fetching components: %w", err)
		}

	case "config_scrapers":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.ConfigScrapers).Error; err != nil {
			return nil, fmt.Errorf("error fetching config scrapers: %w", err)
		}

	case "config_items":
		if err := tx.Not(ids).Where("id::TEXT > ?", paginateReq.From).Limit(paginateReq.Size).Order("id").Find(&pushData.ConfigItems).Error; err != nil {
			return nil, fmt.Errorf("error fetching config items: %w", err)
		}

	case "check_statuses":
		parts := strings.Split(paginateReq.From, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("%s is not a valid next cursor. It must consist of check_id and time separated by a comma", paginateReq.From)
		}

		tx := ctx.DB().Where("(check_id::TEXT, time::TEXT) > (?, ?)", parts[0], parts[1])

		// Attach a Not IN query only if required
		if len(ids) != 0 {
			var pKeys = make([][]string, 0, len(ids))
			for _, pkey := range ids {
				parts := strings.Split(pkey, ",")
				pKeys = append(pKeys, parts)
			}

			tx = tx.Where("(check_id::TEXT, time::TEXT) NOT IN (?)", pKeys)
		}

		if err := tx.Limit(paginateReq.Size).Order("check_id, time").Find(&pushData.CheckStatuses).Error; err != nil {
			return nil, fmt.Errorf("error fetching config items: %w", err)
		}
	}

	return &pushData, nil
}
