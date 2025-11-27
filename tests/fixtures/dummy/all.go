package dummy

import (
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/flanksource/duty/view"
)

var CurrentTime = time.Now()

type DummyData struct {
	People []models.Person
	Agents []models.Agent

	Playbooks    []models.Playbook
	PlaybookRuns []models.PlaybookRun
	Connections  []models.Connection

	Topologies             []models.Topology
	Components             []models.Component
	ComponentRelationships []models.ComponentRelationship

	Configs                      []models.ConfigItem
	ConfigLocations              []models.ConfigLocation
	ConfigRelationships          []models.ConfigRelationship
	ConfigScrapers               []models.ConfigScraper
	ConfigChanges                []models.ConfigChange
	ConfigAnalyses               []models.ConfigAnalysis
	ConfigComponentRelationships []models.ConfigComponentRelationship

	Notifications []models.Notification

	Teams      []models.Team
	Incidents  []models.Incident
	Hypotheses []models.Hypothesis
	Responders []models.Responder
	Evidences  []models.Evidence
	Comments   []models.Comment

	Views      []models.View
	ViewPanels []models.ViewPanel
	ViewTables []ViewGeneratedTable

	Canaries                    []models.Canary
	Checks                      []models.Check
	CheckStatuses               []models.CheckStatus
	CheckComponentRelationships []models.CheckComponentRelationship

	Artifacts    []models.Artifact
	JobHistories []models.JobHistory

	Permissions []models.Permission
	Scopes      []models.Scope
}

func (t *DummyData) Populate(ctx context.Context) error {
	createTime := DummyCreatedAt

	gormDB := ctx.DB()

	if err := gormDB.CreateInBatches(t.People, 100).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			if err := t.Delete(gormDB); err != nil {
				return err
			}
			if err := gormDB.CreateInBatches(t.People, 100).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := gormDB.CreateInBatches(t.Connections, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Agents, 100).Error; err != nil {
		return err
	}
	for i := range t.Topologies {
		t.Topologies[i].UpdatedAt = &createTime
	}
	if err := gormDB.CreateInBatches(t.Topologies, 100).Error; err != nil {
		return err
	}
	for i := range t.Components {
		t.Components[i].UpdatedAt = &createTime
	}
	if err := gormDB.CreateInBatches(t.Components, 100).Error; err != nil {
		return err
	}
	for i := range t.ComponentRelationships {
		t.ComponentRelationships[i].UpdatedAt = createTime
	}
	if err := gormDB.CreateInBatches(t.ComponentRelationships, 100).Error; err != nil {
		return err
	}
	for i := range t.ConfigScrapers {
		t.ConfigScrapers[i].CreatedAt = createTime
	}
	if err := gormDB.CreateInBatches(t.ConfigScrapers, 100).Error; err != nil {
		return err
	}

	for i := range t.Configs {
		t.Configs[i].UpdatedAt = &createTime
	}
	if err := gormDB.CreateInBatches(t.Configs, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.ConfigLocations, 100).Error; err != nil {
		return err
	}

	for i := range t.ConfigRelationships {
		t.ConfigRelationships[i].UpdatedAt = createTime
	}
	if err := gormDB.Model(models.ConfigRelationship{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "related_id"}, {Name: "config_id"}, {Name: "relation"}},
		DoNothing: true,
	}).CreateInBatches(t.ConfigRelationships, 100).Error; err != nil {
		return err
	}
	if err := gormDB.CreateInBatches(t.ConfigChanges, 100).Error; err != nil {
		return err
	}
	for i := range t.ConfigAnalyses {
		t.ConfigAnalyses[i].FirstObserved = &createTime
	}
	if err := gormDB.CreateInBatches(t.ConfigAnalyses, 100).Error; err != nil {
		return err
	}
	if err := gormDB.CreateInBatches(t.ConfigComponentRelationships, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Teams, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Incidents, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Hypotheses, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Evidences, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Responders, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Comments, 100).Error; err != nil {
		return err
	}
	if err := gormDB.CreateInBatches(t.Canaries, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Checks, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.CheckStatuses, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.CheckComponentRelationships, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Playbooks, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.PlaybookRuns, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Artifacts, 100).Error; err != nil {
		return err
	}
	if err := gormDB.CreateInBatches(t.JobHistories, 100).Error; err != nil {
		return err
	}

	if err := gormDB.Exec("UPDATE config_items set path = config_path(id)").Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Notifications, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Scopes, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Permissions, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.Views, 100).Error; err != nil {
		return err
	}

	if err := gormDB.CreateInBatches(t.ViewPanels, 100).Error; err != nil {
		return err
	}

	for _, viewTable := range t.ViewTables {
		columnDefs, err := view.GetViewColumnDefs(ctx, viewTable.View.Namespace, viewTable.View.Name)
		if err != nil {
			return err
		}

		if err := view.CreateViewTable(ctx, viewTable.View.GeneratedTableName(), columnDefs); err != nil {
			return err
		}

		var viewRows []view.Row
		for _, row := range viewTable.Rows {
			viewRow := make(view.Row, len(columnDefs))
			for i, col := range columnDefs {
				if val, exists := row[col.Name]; exists {
					viewRow[i] = val
				}
			}
			viewRows = append(viewRows, viewRow)
		}

		if err := view.InsertViewRows(ctx, viewTable.View.GeneratedTableName(), columnDefs, viewRows, "dummy-fixture"); err != nil {
			return err
		}
	}

	return nil
}

func DeleteAll[T models.DBTable](gormDB *gorm.DB, items []T) error {
	ids := lo.Map(items, func(i T, _ int) string { return i.PK() })
	pk := "id"
	var zero T
	switch any(zero).(type) {
	case models.ViewPanel:
		pk = "view_id"
	}
	return gormDB.Where(pk+" IN (?)", ids).Delete(new(T)).Error
}

func (t *DummyData) Delete(gormDB *gorm.DB) error {

	if err := models.DeleteAllIncidents(gormDB, t.Incidents...); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Permissions); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Scopes); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Artifacts); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.JobHistories); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.PlaybookRuns); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Playbooks); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.ConfigScrapers); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Notifications); err != nil {
		return err
	}

	if err := models.DeleteAllComponents(gormDB, t.Components...); err != nil {
		return err
	}

	if err := models.DeleteAllConfigs(gormDB, t.Configs...); err != nil {
		return err
	}

	if err := models.DeleteAllCanaries(gormDB, t.Canaries...); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Topologies); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Connections); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Teams); err != nil {
		return err
	}
	people_ids := lo.Map(t.People, func(p models.Person, _ int) string { return p.ID.String() })

	if err := gormDB.Exec("DELETE from teams WHERE created_by in (?)", people_ids).Error; err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.People); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Agents); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.ViewPanels); err != nil {
		return err
	}

	if err := DeleteAll(gormDB, t.Views); err != nil {
		return err
	}

	return nil
}

func GetStaticDummyData(db *gorm.DB) DummyData {
	if err := db.Raw("Select now()").Scan(&CurrentTime).Error; err != nil {
		logger.Fatalf("Cannot get current time from db: %v", err)
	}

	// we're appending here so we do not mutate the original slice.
	d := DummyData{
		Playbooks:                    append([]models.Playbook{}, AllDummyPlaybooks...),
		PlaybookRuns:                 append([]models.PlaybookRun{}, AllDummyPlaybookRuns...),
		Connections:                  append([]models.Connection{}, AllDummyConnections...),
		People:                       append([]models.Person{}, AllDummyPeople...),
		Agents:                       append([]models.Agent{}, AllDummyAgents...),
		Topologies:                   append([]models.Topology{}, AllDummyTopologies...),
		Components:                   append([]models.Component{}, AllDummyComponents...),
		ComponentRelationships:       append([]models.ComponentRelationship{}, AllDummyComponentRelationships...),
		ConfigScrapers:               append([]models.ConfigScraper{}, AllConfigScrapers...),
		Configs:                      append([]models.ConfigItem{}, AllDummyConfigs...),
		ConfigLocations:              append([]models.ConfigLocation{}, AllDummyConfigLocations...),
		ConfigChanges:                append([]models.ConfigChange{}, AllDummyConfigChanges...),
		ConfigRelationships:          append([]models.ConfigRelationship{}, AllConfigRelationships...),
		ConfigAnalyses:               append([]models.ConfigAnalysis{}, AllDummyConfigAnalysis()...),
		ConfigComponentRelationships: append([]models.ConfigComponentRelationship{}, AllDummyConfigComponentRelationships...),
		Teams:                        append([]models.Team{}, AllDummyTeams...),
		Notifications:                append([]models.Notification{}, AllDummyNotifications...),
		Incidents:                    append([]models.Incident{}, AllDummyIncidents...),
		Hypotheses:                   append([]models.Hypothesis{}, AllDummyHypotheses...),
		Evidences:                    append([]models.Evidence{}, AllDummyEvidences...),
		Canaries:                     append([]models.Canary{}, AllDummyCanaries...),
		Checks:                       append([]models.Check{}, AllDummyChecks()...),
		CheckStatuses:                append([]models.CheckStatus{}, AllDummyCheckStatuses()...),
		Responders:                   append([]models.Responder{}, AllDummyResponders...),
		Comments:                     append([]models.Comment{}, AllDummyComments...),
		CheckComponentRelationships:  append([]models.CheckComponentRelationship{}, AllDummyCheckComponentRelationships...),
		Artifacts:                    append([]models.Artifact{}, AllDummyArtifacts...),
		Views:                        append([]models.View{}, AllDummyViews...),
		ViewPanels:                   append([]models.ViewPanel{}, AllDummyViewPanels...),
		ViewTables:                   append([]ViewGeneratedTable{}, AllDummyViewTables...),
		JobHistories:                 append([]models.JobHistory{}, AllDummyJobHistories...),
		Permissions:                  append([]models.Permission{}, AllDummyPermissions...),
		Scopes:                       append([]models.Scope{}, AllDummyScopes...),
	}

	return d
}

// GenerateDynamicDummyData is similar to GetStaticDummyData()
// except that the ids are randomly generated on call.
func GenerateDynamicDummyData(db *gorm.DB) DummyData {

	var JohnDoeDynamic = models.Person{
		ID:    uuid.New(),
		Name:  "John Doe",
		Email: "john@doe.com",
	}

	var JohnWickDynamic = models.Person{
		ID:    uuid.New(),
		Name:  "John Wick",
		Email: "john@wick.com",
	}
	if err := db.Raw("Select now()").Scan(&CurrentTime).Error; err != nil {
		logger.Fatalf("Cannot get current time from db: %v", err)
	}

	var (
		DummyCreatedAt   = time.Date(2022, time.December, 31, 23, 59, 0, 0, time.UTC)
		DummyYearOldDate = CurrentTime.AddDate(-1, 0, 0)
	)

	var people = []models.Person{JohnDoeDynamic, JohnWickDynamic}

	// Agents
	var GCPAgent = models.Agent{
		ID:   uuid.New(),
		Name: "GCP",
	}

	var agents = []models.Agent{
		GCPAgent,
	}

	// Teams
	var BackendTeam = models.Team{
		ID:        uuid.New(),
		Name:      "Backend",
		Icon:      "backend",
		CreatedBy: JohnDoeDynamic.ID,
		CreatedAt: CurrentTime,
		UpdatedAt: CurrentTime,
	}

	var FrontendTeam = models.Team{
		ID:        uuid.New(),
		Name:      "Frontend",
		Icon:      "frontend",
		CreatedBy: JohnDoeDynamic.ID,
		CreatedAt: CurrentTime,
		UpdatedAt: CurrentTime,
	}

	var teams = []models.Team{BackendTeam, FrontendTeam}

	// Topologies
	var LogisticsTopology = models.Topology{
		ID:        uuid.New(),
		Name:      "logistics",
		Namespace: "default",
		Source:    models.SourceUI,
	}

	var topologies = []models.Topology{
		LogisticsTopology,
	}

	// Components
	var Logistics = models.Component{
		ID:         uuid.New(),
		Name:       "logistics",
		Type:       "Entity",
		ExternalId: "dummy/logistics",
		Labels:     types.JSONStringMap{"telemetry": "enabled"},
		Owner:      "logistics-team",
		CreatedAt:  DummyCreatedAt,
		Status:     types.ComponentStatusHealthy,
	}

	var LogisticsAPI = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-api",
		ExternalId: "dummy/logistics-api",
		Type:       "Application",
		Status:     types.ComponentStatusHealthy,
		Labels:     types.JSONStringMap{"telemetry": "enabled"},
		Owner:      "logistics-team",
		ParentId:   &Logistics.ID,
		Path:       Logistics.ID.String(),
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsUI = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-ui",
		Type:       "Application",
		ExternalId: "dummy/logistics-ui",
		Status:     types.ComponentStatusHealthy,
		Owner:      "logistics-team",
		ParentId:   &Logistics.ID,
		Path:       Logistics.ID.String(),
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsWorker = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-worker",
		ExternalId: "dummy/logistics-worker",
		Type:       "Application",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &LogisticsAPI.ID,
		Path:       Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsDB = models.Component{
		ID:           uuid.New(),
		Name:         "logistics-db",
		ExternalId:   "dummy/logistics-db",
		Type:         "Database",
		Status:       types.ComponentStatusUnhealthy,
		StatusReason: "database not accepting connections",
		ParentId:     &LogisticsAPI.ID,
		Path:         Logistics.ID.String() + "." + LogisticsAPI.ID.String(),
		CreatedAt:    DummyCreatedAt,
	}

	var ClusterComponent = models.Component{
		ID:         uuid.New(),
		Name:       "cluster",
		ExternalId: "dummy/cluster",
		Type:       "KubernetesCluster",
		Status:     types.ComponentStatusHealthy,
		CreatedAt:  DummyCreatedAt,
	}

	var NodesComponent = models.Component{
		ID:         uuid.New(),
		Name:       "Nodes",
		ExternalId: "dummy/nodes",
		Type:       "KubernetesNodes",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &ClusterComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var NodeA = models.Component{
		ID:         uuid.New(),
		Name:       "node-a",
		ExternalId: "dummy/node-a",
		Type:       "KubernetesNode",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &NodesComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var NodeB = models.Component{
		ID:         uuid.New(),
		Name:       "node-b",
		ExternalId: "dummy/node-b",
		Type:       "KubernetesNode",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &NodesComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var PodsComponent = models.Component{
		ID:         uuid.New(),
		Name:       "Pods",
		ExternalId: "dummy/pods",
		Type:       "KubernetesPods",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &ClusterComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsAPIPod = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-api-574dc95b5d-mp64w",
		ExternalId: "dummy/logistics-api-574dc95b5d-mp64w",
		Type:       "KubernetesPod",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &PodsComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsUIPod = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-ui-676b85b87c-tjjcp",
		Type:       "KubernetesPod",
		ExternalId: "dummy/logistics-ui-676b85b87c-tjjcp",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &PodsComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var LogisticsWorkerPod = models.Component{
		ID:         uuid.New(),
		Name:       "logistics-worker-79cb67d8f5-lr66n",
		ExternalId: "dummy/logistics-worker-79cb67d8f5-lr66n",
		Type:       "KubernetesPod",
		Status:     types.ComponentStatusHealthy,
		ParentId:   &PodsComponent.ID,
		CreatedAt:  DummyCreatedAt,
	}

	var PaymentsAPI = models.Component{
		ID:         uuid.New(),
		AgentID:    GCPAgent.ID,
		Name:       "payments-api",
		ExternalId: "dummy/payments-api",
		Type:       "Application",
		CreatedAt:  DummyCreatedAt,
		Status:     types.ComponentStatusHealthy,
	}

	// Order is important since ParentIDs refer to previous components
	var components = []models.Component{
		Logistics,
		LogisticsAPI,
		LogisticsUI,
		LogisticsWorker,
		LogisticsDB,
		ClusterComponent,
		NodesComponent,
		PodsComponent,
		NodeA,
		NodeB,
		LogisticsAPIPod,
		LogisticsUIPod,
		LogisticsWorkerPod,
		PaymentsAPI,
	}

	// Canaries
	var LogisticsAPICanary = models.Canary{
		ID:        uuid.New(),
		Name:      "dummy-logistics-api-canary",
		Namespace: "logistics",
		Spec:      []byte("{}"),
		CreatedAt: DummyCreatedAt,
	}

	var LogisticsDBCanary = models.Canary{
		ID:        uuid.New(),
		Name:      "dummy-logistics-db-canary",
		Namespace: "logistics",
		Spec:      []byte("{}"),
		CreatedAt: DummyCreatedAt,
	}

	var CartAPICanaryAgent = models.Canary{
		ID:        uuid.New(),
		AgentID:   GCPAgent.ID,
		Name:      "dummy-cart-api-canary",
		Namespace: "cart",
		Spec:      []byte("{}"),
		CreatedAt: DummyCreatedAt,
	}

	var canaries = []models.Canary{LogisticsAPICanary, LogisticsDBCanary, CartAPICanaryAgent}

	// Checks
	var LogisticsAPIHealthHTTPCheck = models.Check{
		ID:       uuid.New(),
		CanaryID: LogisticsAPICanary.ID,
		Name:     "logistics-api-health-check",
		Type:     "http",
		Status:   "healthy",
	}

	var LogisticsAPIHomeHTTPCheck = models.Check{
		ID:       uuid.New(),
		CanaryID: LogisticsAPICanary.ID,
		Name:     "logistics-api-home-check",
		Type:     "http",
		Status:   "healthy",
	}

	var LogisticsDBCheck = models.Check{
		ID:       uuid.New(),
		CanaryID: LogisticsDBCanary.ID,
		Name:     "logistics-db-check",
		Type:     "postgres",
		Status:   "unhealthy",
	}

	var CartAPIHeathCheckAgent = models.Check{
		ID:       uuid.New(),
		AgentID:  GCPAgent.ID,
		CanaryID: CartAPICanaryAgent.ID,
		Name:     "cart-api-health-check",
		Type:     "http",
		Status:   models.CheckHealthStatus(types.ComponentStatusHealthy),
	}

	var checks = []models.Check{
		LogisticsAPIHealthHTTPCheck,
		LogisticsAPIHomeHTTPCheck,
		LogisticsDBCheck,
		CartAPIHeathCheckAgent,
	}

	// Check statuses
	var t1 = CurrentTime.Add(-15 * time.Minute)
	var t2 = CurrentTime.Add(-10 * time.Minute)
	var t3 = CurrentTime.Add(-5 * time.Minute)

	var LogisticsAPIHealthHTTPCheckStatus1 = models.CheckStatus{
		CheckID:   LogisticsAPIHealthHTTPCheck.ID,
		Duration:  100,
		Status:    true,
		CreatedAt: t1,
		Time:      t1.Format("2006-01-02 15:04:05"),
	}

	var LogisticsAPIHealthHTTPCheckStatus2 = models.CheckStatus{
		CheckID:   LogisticsAPIHealthHTTPCheck.ID,
		Duration:  100,
		Status:    true,
		CreatedAt: t2,
		Time:      t2.Format("2006-01-02 15:04:05"),
	}

	var LogisticsAPIHealthHTTPCheckStatus3 = models.CheckStatus{
		CheckID:   LogisticsAPIHealthHTTPCheck.ID,
		Duration:  100,
		Status:    true,
		CreatedAt: t3,
		Time:      t3.Format("2006-01-02 15:04:05"),
	}

	var LogisticsAPIHomeHTTPCheckStatus1 = models.CheckStatus{
		CheckID:   LogisticsAPIHomeHTTPCheck.ID,
		Duration:  100,
		Status:    true,
		CreatedAt: t1,
		Time:      t3.Format("2006-01-02 15:04:05"),
	}

	var LogisticsDBCheckStatus1 = models.CheckStatus{
		CheckID:   LogisticsDBCheck.ID,
		Duration:  50,
		Status:    false,
		CreatedAt: t1,
		Time:      t1.Format("2006-01-02 15:04:05"),
	}

	var checkStatuses = []models.CheckStatus{
		LogisticsAPIHealthHTTPCheckStatus1,
		LogisticsAPIHealthHTTPCheckStatus2,
		LogisticsAPIHealthHTTPCheckStatus3,
		LogisticsAPIHomeHTTPCheckStatus1,
		LogisticsDBCheckStatus1,
	}

	// Config scrapers
	var AzureConfigScraper = models.ConfigScraper{
		ID:     uuid.New(),
		Name:   "Azure scraper",
		Source: "ConfigFile",
		Spec:   "{}",
	}

	var configScrapers = []models.ConfigScraper{AzureConfigScraper}

	var EKSCluster = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassCluster,
		Type:        lo.ToPtr("EKS::Cluster"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"telemetry":   "enabled",
			"environment": "production",
		}),
	}

	var KubernetesCluster = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassCluster,
		Type:        lo.ToPtr("Kubernetes::Cluster"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"telemetry":   "enabled",
			"environment": "development",
		}),
	}

	var KubernetesNodeA = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassNode,
		Type:        lo.ToPtr("Kubernetes::Node"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"role":   "worker",
			"region": "us-east-1",
		}),
		CostTotal30d: 1,
	}

	var KubernetesNodeB = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassNode,
		Type:        lo.ToPtr("Kubernetes::Node"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"role":           "worker",
			"region":         "us-west-2",
			"storageprofile": "managed",
		}),
		CostTotal30d: 1.5,
	}

	var EC2InstanceA = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassVirtualMachine,
		Type:        lo.ToPtr("EC2::Instance"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"environment": "testing",
			"app":         "backend",
		}),
	}

	var EC2InstanceB = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassVirtualMachine,
		Type:        lo.ToPtr("EC2::Instance"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"environment": "production",
			"app":         "frontend",
		}),
	}

	var LogisticsAPIDeployment = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassDeployment,
		Type:        lo.ToPtr("Logistics::API::Deployment"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"app":         "logistics",
			"environment": "production",
			"owner":       "team-1",
			"version":     "1.2.0",
		}),
	}

	var LogisticsUIDeployment = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassDeployment,
		Type:        lo.ToPtr("Logistics::UI::Deployment"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"app":         "logistics",
			"environment": "production",
			"owner":       "team-2",
			"version":     "2.0.1",
		}),
	}

	var LogisticsWorkerDeployment = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassDeployment,
		Type:        lo.ToPtr("Logistics::Worker::Deployment"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"app":         "logistics",
			"environment": "production",
			"owner":       "team-3",
			"version":     "1.5.0",
		}),
	}

	var LogisticsDBRDS = models.ConfigItem{
		ID:          uuid.New(),
		ConfigClass: models.ConfigClassDatabase,
		Type:        lo.ToPtr("Logistics::DB::RDS"),
		Labels: lo.ToPtr(types.JSONStringMap{
			"database":    "logistics",
			"environment": "production",
			"region":      "us-east-1",
			"size":        "large",
		}),
	}

	var configs = []models.ConfigItem{
		EKSCluster,
		KubernetesCluster,
		KubernetesNodeA,
		KubernetesNodeB,
		EC2InstanceA,
		EC2InstanceB,
		LogisticsAPIDeployment,
		LogisticsUIDeployment,
		LogisticsWorkerDeployment,
		LogisticsDBRDS,
	}

	var ClusterNodeARelationship = models.ConfigRelationship{
		ConfigID:  KubernetesCluster.ID.String(),
		RelatedID: KubernetesNodeA.ID.String(),
		Relation:  "ClusterNode",
		CreatedAt: DummyCreatedAt,
	}

	var ClusterNodeBRelationship = models.ConfigRelationship{
		ConfigID:  KubernetesCluster.ID.String(),
		RelatedID: KubernetesNodeB.ID.String(),
		Relation:  "ClusterNode",
		CreatedAt: DummyCreatedAt,
	}

	var configRelationships = []models.ConfigRelationship{ClusterNodeARelationship, ClusterNodeBRelationship}

	var LogisticsDBRDSAnalysis = models.ConfigAnalysis{
		ID:            uuid.New(),
		ConfigID:      LogisticsDBRDS.ID,
		AnalysisType:  models.AnalysisTypeSecurity,
		Severity:      "critical",
		Message:       "Port exposed to public",
		FirstObserved: &CurrentTime,
		Status:        models.AnalysisStatusOpen,
	}

	var EC2InstanceBAnalysis = models.ConfigAnalysis{
		ID:            uuid.New(),
		ConfigID:      EC2InstanceB.ID,
		AnalysisType:  models.AnalysisTypeSecurity,
		Severity:      "critical",
		Message:       "SSH key not rotated",
		FirstObserved: &CurrentTime,
		Status:        models.AnalysisStatusOpen,
	}

	var configAnalysis = []models.ConfigAnalysis{
		LogisticsDBRDSAnalysis,
		EC2InstanceBAnalysis,
	}

	var EKSClusterCreateChange = models.ConfigChange{
		ID:         uuid.New().String(),
		ConfigID:   EKSCluster.ID.String(),
		ChangeType: "CREATE",
		CreatedAt:  &DummyYearOldDate,
	}

	var EKSClusterUpdateChange = models.ConfigChange{
		ID:         uuid.New().String(),
		ConfigID:   EKSCluster.ID.String(),
		ChangeType: "UPDATE",
	}

	var EKSClusterDeleteChange = models.ConfigChange{
		ID:         uuid.New().String(),
		ConfigID:   EKSCluster.ID.String(),
		ChangeType: "DELETE",
	}

	var KubernetesNodeAChange = models.ConfigChange{
		ID:         uuid.New().String(),
		ConfigID:   KubernetesNodeA.ID.String(),
		ChangeType: "CREATE",
	}

	var configChanges = []models.ConfigChange{
		EKSClusterCreateChange,
		EKSClusterUpdateChange,
		EKSClusterDeleteChange,
		KubernetesNodeAChange,
	}

	// Incidents
	var LogisticsAPIDownIncident = models.Incident{
		ID:          uuid.New(),
		Title:       "Logistics API is down",
		CreatedBy:   JohnDoeDynamic.ID,
		Type:        models.IncidentTypeAvailability,
		Status:      models.IncidentStatusOpen,
		Severity:    "Blocker",
		CommanderID: &JohnDoeDynamic.ID,
	}

	var UIDownIncident = models.Incident{
		ID:          uuid.New(),
		Title:       "UI is down",
		CreatedBy:   JohnDoeDynamic.ID,
		Type:        models.IncidentTypeAvailability,
		Status:      models.IncidentStatusOpen,
		Severity:    "Blocker",
		CommanderID: &JohnWickDynamic.ID,
	}

	var incidents = []models.Incident{LogisticsAPIDownIncident, UIDownIncident}

	// Hypotheses
	var LogisticsAPIDownHypothesis = models.Hypothesis{
		ID:         uuid.New(),
		IncidentID: LogisticsAPIDownIncident.ID,
		Title:      "Logistics DB database error hypothesis",
		CreatedBy:  JohnDoeDynamic.ID,
		Type:       "solution",
		Status:     "possible",
	}

	var hypotheses = []models.Hypothesis{LogisticsAPIDownHypothesis}

	// Evidences
	var LogisticsDBErrorEvidence = models.Evidence{
		ID:           uuid.New(),
		HypothesisID: LogisticsAPIDownHypothesis.ID,
		ComponentID:  &LogisticsDB.ID,
		CreatedBy:    JohnDoeDynamic.ID,
		Description:  "Logisctics DB attached component",
		Type:         "component",
	}

	var evidences = []models.Evidence{LogisticsDBErrorEvidence}

	// Comments
	var FirstComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnWickDynamic.ID,
		Comment:    "This is a comment",
		IncidentID: LogisticsAPIDownIncident.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var SecondComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnDoeDynamic.ID,
		Comment:    "A comment by John Doe",
		IncidentID: LogisticsAPIDownIncident.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var ThirdComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnDoeDynamic.ID,
		Comment:    "Another comment by John Doe",
		IncidentID: LogisticsAPIDownIncident.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var comments = []models.Comment{FirstComment, SecondComment, ThirdComment}

	// Responders
	var JiraResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: LogisticsAPIDownIncident.ID,
		Type:       "Jira",
		PersonID:   &JohnWickDynamic.ID,
		CreatedBy:  JohnWickDynamic.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var GitHubIssueResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: LogisticsAPIDownIncident.ID,
		Type:       "GithubIssue",
		PersonID:   &JohnDoeDynamic.ID,
		CreatedBy:  JohnDoeDynamic.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var SlackResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "Slack",
		TeamID:     &BackendTeam.ID,
		CreatedBy:  JohnDoeDynamic.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var MsPlannerResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "MSPlanner",
		PersonID:   &JohnWickDynamic.ID,
		CreatedBy:  JohnDoeDynamic.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var TelegramResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "Telegram",
		PersonID:   &JohnDoeDynamic.ID,
		CreatedBy:  JohnDoeDynamic.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var responders = []models.Responder{JiraResponder, GitHubIssueResponder, SlackResponder, MsPlannerResponder, TelegramResponder}

	// CheckComponentRelationship
	var LogisticsDBCheckComponentRelationship = models.CheckComponentRelationship{
		ComponentID: LogisticsDB.ID,
		CheckID:     LogisticsDBCheck.ID,
		CanaryID:    LogisticsDBCheck.CanaryID,
	}

	var LogisticsAPIHealthHTTPCheckComponentRelationship = models.CheckComponentRelationship{
		ComponentID: LogisticsAPI.ID,
		CheckID:     LogisticsAPIHealthHTTPCheck.ID,
		CanaryID:    LogisticsAPIHealthHTTPCheck.CanaryID,
	}

	var LogisticsAPIHomeHTTPCheckComponentRelationship = models.CheckComponentRelationship{
		ComponentID: LogisticsAPI.ID,
		CheckID:     LogisticsAPIHomeHTTPCheck.ID,
		CanaryID:    LogisticsAPIHomeHTTPCheck.CanaryID,
	}

	var checkComponentRelationships = []models.CheckComponentRelationship{
		LogisticsDBCheckComponentRelationship,
		LogisticsAPIHealthHTTPCheckComponentRelationship,
		LogisticsAPIHomeHTTPCheckComponentRelationship,
	}

	// ConfigComponentRelationship
	var EKSClusterClusterComponentRelationship = models.ConfigComponentRelationship{
		ConfigID:    EKSCluster.ID,
		ComponentID: ClusterComponent.ID,
	}

	var KubernetesClusterClusterComponentRelationship = models.ConfigComponentRelationship{
		ConfigID:    KubernetesCluster.ID,
		ComponentID: ClusterComponent.ID,
	}

	var LogisticsDBRDSLogisticsDBComponentRelationship = models.ConfigComponentRelationship{
		ConfigID:    LogisticsDBRDS.ID,
		ComponentID: LogisticsDB.ID,
	}

	var EC2InstanceBNodeBRelationship = models.ConfigComponentRelationship{
		ConfigID:    EC2InstanceB.ID,
		ComponentID: NodeB.ID,
	}

	var configComponentRelationships = []models.ConfigComponentRelationship{
		EKSClusterClusterComponentRelationship,
		KubernetesClusterClusterComponentRelationship,
		LogisticsDBRDSLogisticsDBComponentRelationship,
		EC2InstanceBNodeBRelationship,
	}

	// Component relationships
	var LogisticsAPIPodNodeAComponentRelationship = models.ComponentRelationship{
		ComponentID:    LogisticsAPIPod.ID,
		RelationshipID: NodeA.ID,
	}

	var LogisticsUIPodNodeAComponentRelationship = models.ComponentRelationship{
		ComponentID:    LogisticsUIPod.ID,
		RelationshipID: NodeA.ID,
	}

	var LogisticsWorkerPodNodeBComponentRelationship = models.ComponentRelationship{
		ComponentID:    LogisticsWorkerPod.ID,
		RelationshipID: NodeB.ID,
	}

	var componentRelationships = []models.ComponentRelationship{
		LogisticsAPIPodNodeAComponentRelationship,
		LogisticsUIPodNodeAComponentRelationship,
		LogisticsWorkerPodNodeBComponentRelationship,
	}

	d := DummyData{
		People: people,
		Agents: agents,

		Topologies:             topologies,
		Components:             components,
		ComponentRelationships: componentRelationships,

		ConfigRelationships:          configRelationships,
		ConfigScrapers:               configScrapers,
		Configs:                      configs,
		ConfigChanges:                configChanges,
		ConfigAnalyses:               configAnalysis,
		ConfigComponentRelationships: configComponentRelationships,

		Teams:      teams,
		Responders: responders,
		Incidents:  incidents,
		Hypotheses: hypotheses,
		Evidences:  evidences,
		Comments:   comments,

		Canaries:                    canaries,
		Checks:                      checks,
		CheckStatuses:               checkStatuses,
		CheckComponentRelationships: checkComponentRelationships,
	}

	return d
}
