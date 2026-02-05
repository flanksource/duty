package upstream

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flanksource/commons/collections"
	"github.com/flanksource/commons/http"
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
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

func (t *UpstreamConfig) IsPartiallyFilled() (bool, error) {
	isPartial := !t.Valid() && (t.Host != "" || t.Password != "" || t.AgentName != "")
	if !isPartial {
		return false, nil
	}

	var errorMsg string
	if t.Host == "" {
		errorMsg += "UPSTREAM_HOST is empty."
	}
	if t.Password == "" {
		errorMsg += "UPSTREAM_PASSWORD is empty."
	}
	if t.AgentName == "" {
		errorMsg += "AGENT_NAME is empty."
	}

	return true, fmt.Errorf("%s", errorMsg)
}

func (t *UpstreamConfig) LabelsMap() map[string]string {
	return collections.KeyValueSliceToMap(t.Labels)
}

// PushData consists of data about changes to
// components, configs, analysis.
type PushData struct {
	Canaries                     []models.Canary                        `json:"canaries,omitempty"`
	Checks                       []models.Check                         `json:"checks,omitempty"`
	Components                   []models.Component                     `json:"components,omitempty"`
	ConfigScrapers               []models.ConfigScraper                 `json:"config_scrapers,omitempty"`
	ConfigAnalysis               []models.ConfigAnalysis                `json:"config_analysis,omitempty"`
	ConfigChanges                []models.ConfigChange                  `json:"config_changes,omitempty"`
	ConfigItems                  []models.ConfigItem                    `json:"config_items,omitempty"`
	CheckStatuses                []models.CheckStatus                   `json:"check_statuses,omitempty"`
	ConfigRelationships          []models.ConfigRelationship            `json:"config_relationships,omitempty"`
	ComponentRelationships       []models.ComponentRelationship         `json:"component_relationships,omitempty"`
	ConfigComponentRelationships []models.ConfigComponentRelationship   `json:"config_component_relationships,omitempty"`
	Topologies                   []models.Topology                      `json:"topologies,omitempty"`
	PlaybookActions              []models.PlaybookRunAction             `json:"playbook_actions,omitempty"`
	Artifacts                    []models.Artifact                      `json:"artifacts,omitempty"`
	JobHistory                   []models.JobHistory                    `json:"job_history,omitempty"`
	ViewPanels                   []models.ViewPanel                     `json:"view_panels,omitempty"`
	GeneratedViews               map[string][]models.GeneratedViewTable `json:"generated_views,omitempty"`
}

func NewPushData[T models.DBTable](records []T) *PushData {
	var p PushData
	if len(records) == 0 {
		return &p
	}

	p.GeneratedViews = make(map[string][]models.GeneratedViewTable)

	for i := range records {
		switch t := any(records[i]).(type) {
		case models.Canary:
			p.Canaries = append(p.Canaries, t)
		case models.Check:
			p.Checks = append(p.Checks, t)
		case models.Component:
			p.Components = append(p.Components, t)
		case models.ConfigScraper:
			p.ConfigScrapers = append(p.ConfigScrapers, t)
		case models.ConfigAnalysis:
			p.ConfigAnalysis = append(p.ConfigAnalysis, t)
		case models.ConfigChange:
			p.ConfigChanges = append(p.ConfigChanges, t)
		case models.ConfigItem:
			p.ConfigItems = append(p.ConfigItems, t)
		case models.CheckStatus:
			p.CheckStatuses = append(p.CheckStatuses, t)
		case models.ConfigRelationship:
			p.ConfigRelationships = append(p.ConfigRelationships, t)
		case models.ComponentRelationship:
			p.ComponentRelationships = append(p.ComponentRelationships, t)
		case models.ConfigComponentRelationship:
			p.ConfigComponentRelationships = append(p.ConfigComponentRelationships, t)
		case models.Topology:
			p.Topologies = append(p.Topologies, t)
		case models.PlaybookRunAction:
			p.PlaybookActions = append(p.PlaybookActions, t)
		case models.Artifact:
			p.Artifacts = append(p.Artifacts, t)
		case models.JobHistory:
			p.JobHistory = append(p.JobHistory, t)
		case models.ViewPanel:
			p.ViewPanels = append(p.ViewPanels, t)
		case models.GeneratedViewTable:
			p.GeneratedViews[t.ViewTableName] = append(p.GeneratedViews[t.ViewTableName], t)
		}
	}

	return &p
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
	counter.Label("table", "job_history").Add(len(p.JobHistory))
	counter.Label("table", "view_panels").Add(len(p.ViewPanels))

	for tableName := range p.GeneratedViews {
		counter.Label("table", tableName).Add(len(p.GeneratedViews[tableName]))
	}
}

func (p *PushData) String() string {
	result := ""
	for k, v := range p.Attributes() {
		result += fmt.Sprintf("%s=%v ", k, v)
	}
	return strings.TrimSpace(result)
}

func (p *PushData) Attributes() map[string]any {
	attrs := map[string]any{}

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
	if len(p.JobHistory) > 0 {
		attrs["JobHistory"] = len(p.JobHistory)
	}
	if len(p.ViewPanels) > 0 {
		attrs["ViewPanels"] = len(p.ViewPanels)
	}
	if len(p.GeneratedViews) > 0 {
		attrs["ViewData"] = len(p.GeneratedViews)
	}

	return attrs
}

// Size returns the size of JSON encoded pushdata in bytes
func (t PushData) Size() (int, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (t *PushData) Count() int {
	return len(t.Canaries) + len(t.Checks) + len(t.Components) + len(t.ConfigScrapers) +
		len(t.ConfigAnalysis) + len(t.ConfigChanges) + len(t.ConfigItems) + len(t.CheckStatuses) +
		len(t.ConfigRelationships) + len(t.ComponentRelationships) + len(t.ConfigComponentRelationships) +
		len(t.Topologies) + len(t.PlaybookActions) + len(t.Artifacts) + len(t.JobHistory) +
		len(t.ViewPanels) + len(t.GeneratedViews)
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
	for i := range t.JobHistory {
		t.JobHistory[i].AgentID = id
	}
	for i := range t.ViewPanels {
		t.ViewPanels[i].AgentID = id
	}
	for view := range t.GeneratedViews {
		for row := range t.GeneratedViews[view] {
			t.GeneratedViews[view][row].Row["agent_id"] = id
		}
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

func (t *PushData) AddAgentConfig(agent models.Agent) {
	// Filter out local agent config if present
	t.ConfigItems = lo.Filter(t.ConfigItems, func(cs models.ConfigItem, _ int) bool { return cs.ID != uuid.Nil })

	for i, ci := range t.ConfigItems {
		if lo.FromPtr(ci.Type) == "MissionControl::Agent" {
			t.ConfigItems[i].ID = agent.ID
			t.ConfigItems[i].Name = lo.ToPtr(agent.Name)
			t.ConfigItems[i].ScraperID = lo.ToPtr(uuid.Nil.String())
		}
	}

	// Update agent's config changes with correct config ID
	for i, ci := range t.ConfigChanges {
		if ci.ConfigID == uuid.Nil.String() {
			t.ConfigChanges[i].ConfigID = agent.ID.String()
		}
	}

	// Filter out system scraper if present
	t.ConfigScrapers = lo.Filter(t.ConfigScrapers, func(cs models.ConfigScraper, _ int) bool { return cs.ID != uuid.Nil })
}
