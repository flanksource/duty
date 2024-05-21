package dummy

import (
	"fmt"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/utils"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

var CurrentTime = time.Now()

type DummyData struct {
	People []models.Person
	Agents []models.Agent

	Topologies             []models.Topology
	Components             []models.Component
	ComponentRelationships []models.ComponentRelationship

	Configs                      []models.ConfigItem
	ConfigRelationships          []models.ConfigRelationship
	ConfigScrapers               []models.ConfigScraper
	ConfigChanges                []models.ConfigChange
	ConfigAnalyses               []models.ConfigAnalysis
	ConfigComponentRelationships []models.ConfigComponentRelationship

	Teams      []models.Team
	Incidents  []models.Incident
	Hypotheses []models.Hypothesis
	Responders []models.Responder
	Evidences  []models.Evidence
	Comments   []models.Comment

	Canaries                    []models.Canary
	Checks                      []models.Check
	CheckStatuses               []models.CheckStatus
	CheckComponentRelationships []models.CheckComponentRelationship

	Artifacts    []models.Artifact
	JobHistories []models.JobHistory
}

func (t *DummyData) Populate(gormDB *gorm.DB) error {
	var err error
	createTime := DummyCreatedAt
	for _, c := range t.People {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Agents {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Topologies {
		c.UpdatedAt = &createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Components {
		c.UpdatedAt = &createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ComponentRelationships {
		c.UpdatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigScrapers {
		c.CreatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Configs {
		c.CreatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigRelationships {
		c.CreatedAt = createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigChanges {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigAnalyses {
		c.FirstObserved = &createTime
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigComponentRelationships {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Teams {
		if err := gormDB.Create(&c).Error; err != nil {
			return err
		}
	}
	for _, c := range t.Incidents {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Hypotheses {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Evidences {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Canaries {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Checks {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Responders {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy responder: %w", err)
		}
	}
	for _, c := range t.Comments {
		err = gormDB.Create(&c).Error
		if err != nil {
			return fmt.Errorf("error creating dummy comment: %w", err)
		}
	}
	for _, c := range t.CheckStatuses {
		// TODO: Figure out why it panics without Table
		err = gormDB.Table("check_statuses").Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.CheckComponentRelationships {
		err = gormDB.Create(&c).Error
		if err != nil {
			return err
		}
	}
	for _, a := range t.Artifacts {
		err = gormDB.Create(&a).Error
		if err != nil {
			return err
		}
	}
	for _, j := range t.JobHistories {
		err = gormDB.Create(&j).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *DummyData) Delete(gormDB *gorm.DB) error {
	var err error
	if err = gormDB.Exec(`DELETE FROM incident_histories`).Error; err != nil {
		return err
	}

	for _, c := range t.Evidences {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Hypotheses {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Comments {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Responders {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Incidents {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, a := range t.Artifacts {
		err = gormDB.Delete(&a).Error
		if err != nil {
			return err
		}
	}
	for _, j := range t.JobHistories {
		err = gormDB.Delete(&j).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.CheckComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigAnalyses {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Teams {
		if err := gormDB.Delete(&c).Error; err != nil {
			return err
		}
	}
	for _, c := range t.ConfigChanges {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Configs {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ConfigScrapers {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.ComponentRelationships {
		err = gormDB.Where("component_id = ?", c.ComponentID).Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for i := range t.Components {
		// We need to delete in reverse order
		elem := t.Components[len(t.Components)-1-i]
		err = gormDB.Delete(&elem).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Canaries {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Topologies {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.People {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	for _, c := range t.Agents {
		err = gormDB.Delete(&c).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func GetStaticDummyData(db *gorm.DB) DummyData {
	if err := db.Raw("Select now()").Scan(&CurrentTime).Error; err != nil {
		logger.Fatalf("Cannot get current time from db: %v", err)
	}

	// we're appending here so we do not mutate the original slice.
	d := DummyData{
		People:                       append([]models.Person{}, AllDummyPeople...),
		Agents:                       append([]models.Agent{}, AllDummyAgents...),
		Topologies:                   append([]models.Topology{}, AllDummyTopologies...),
		Components:                   append([]models.Component{}, AllDummyComponents...),
		ComponentRelationships:       append([]models.ComponentRelationship{}, AllDummyComponentRelationships...),
		Configs:                      append([]models.ConfigItem{}, AllDummyConfigs...),
		ConfigChanges:                append([]models.ConfigChange{}, AllDummyConfigChanges...),
		ConfigRelationships:          append([]models.ConfigRelationship{}, AllConfigRelationships...),
		ConfigAnalyses:               append([]models.ConfigAnalysis{}, AllDummyConfigAnalysis()...),
		ConfigComponentRelationships: append([]models.ConfigComponentRelationship{}, AllDummyConfigComponentRelationships...),
		Teams:                        append([]models.Team{}, AllDummyTeams...),
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
		JobHistories:                 append([]models.JobHistory{}, AllDummyJobHistories...),
	}

	return d
}

// GenerateDynamicDummyData is similar to GetStaticDummyData()
// except that the ids are randomly generated on call.
func GenerateDynamicDummyData(db *gorm.DB) DummyData {

	if err := db.Raw("Select now()").Scan(&CurrentTime).Error; err != nil {
		logger.Fatalf("Cannot get current time from db: %v", err)
	}

	var (
		DummyCreatedAt   = time.Date(2022, time.December, 31, 23, 59, 0, 0, time.UTC)
		DummyYearOldDate = CurrentTime.AddDate(-1, 0, 0)
	)

	// People
	var JohnDoe = models.Person{
		ID:    uuid.New(),
		Name:  "John Doe",
		Email: "john@doe.com",
	}

	var JohnWick = models.Person{
		ID:    uuid.New(),
		Name:  "John Wick",
		Email: "john@wick.com",
	}

	var people = []models.Person{JohnDoe, JohnWick}

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
		CreatedBy: JohnDoe.ID,
		CreatedAt: CurrentTime,
		UpdatedAt: CurrentTime,
	}

	var FrontendTeam = models.Team{
		ID:        uuid.New(),
		Name:      "Frontend",
		Icon:      "frontend",
		CreatedBy: JohnDoe.ID,
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
		ID:               uuid.New().String(),
		ConfigID:         EKSCluster.ID.String(),
		ChangeType:       "CREATE",
		ExternalChangeId: utils.RandomString(10),
		CreatedAt:        &DummyYearOldDate,
	}

	var EKSClusterUpdateChange = models.ConfigChange{
		ID:               uuid.New().String(),
		ConfigID:         EKSCluster.ID.String(),
		ChangeType:       "UPDATE",
		ExternalChangeId: utils.RandomString(10),
	}

	var EKSClusterDeleteChange = models.ConfigChange{
		ID:               uuid.New().String(),
		ConfigID:         EKSCluster.ID.String(),
		ChangeType:       "DELETE",
		ExternalChangeId: utils.RandomString(10),
	}

	var KubernetesNodeAChange = models.ConfigChange{
		ID:               uuid.New().String(),
		ConfigID:         KubernetesNodeA.ID.String(),
		ChangeType:       "CREATE",
		ExternalChangeId: utils.RandomString(10),
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
		CreatedBy:   JohnDoe.ID,
		Type:        models.IncidentTypeAvailability,
		Status:      models.IncidentStatusOpen,
		Severity:    "Blocker",
		CommanderID: &JohnDoe.ID,
	}

	var UIDownIncident = models.Incident{
		ID:          uuid.New(),
		Title:       "UI is down",
		CreatedBy:   JohnDoe.ID,
		Type:        models.IncidentTypeAvailability,
		Status:      models.IncidentStatusOpen,
		Severity:    "Blocker",
		CommanderID: &JohnWick.ID,
	}

	var incidents = []models.Incident{LogisticsAPIDownIncident, UIDownIncident}

	// Hypotheses
	var LogisticsAPIDownHypothesis = models.Hypothesis{
		ID:         uuid.New(),
		IncidentID: LogisticsAPIDownIncident.ID,
		Title:      "Logistics DB database error hypothesis",
		CreatedBy:  JohnDoe.ID,
		Type:       "solution",
		Status:     "possible",
	}

	var hypotheses = []models.Hypothesis{LogisticsAPIDownHypothesis}

	// Evidences
	var LogisticsDBErrorEvidence = models.Evidence{
		ID:           uuid.New(),
		HypothesisID: LogisticsAPIDownHypothesis.ID,
		ComponentID:  &LogisticsDB.ID,
		CreatedBy:    JohnDoe.ID,
		Description:  "Logisctics DB attached component",
		Type:         "component",
	}

	var evidences = []models.Evidence{LogisticsDBErrorEvidence}

	// Comments
	var FirstComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnWick.ID,
		Comment:    "This is a comment",
		IncidentID: LogisticsAPIDownIncident.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var SecondComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnDoe.ID,
		Comment:    "A comment by John Doe",
		IncidentID: LogisticsAPIDownIncident.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var ThirdComment = models.Comment{
		ID:         uuid.New(),
		CreatedBy:  JohnDoe.ID,
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
		PersonID:   &JohnWick.ID,
		CreatedBy:  JohnWick.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var GitHubIssueResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: LogisticsAPIDownIncident.ID,
		Type:       "GithubIssue",
		PersonID:   &JohnDoe.ID,
		CreatedBy:  JohnDoe.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var SlackResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "Slack",
		TeamID:     &BackendTeam.ID,
		CreatedBy:  JohnDoe.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var MsPlannerResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "MSPlanner",
		PersonID:   &JohnWick.ID,
		CreatedBy:  JohnDoe.ID,
		CreatedAt:  CurrentTime,
		UpdatedAt:  CurrentTime,
	}

	var TelegramResponder = models.Responder{
		ID:         uuid.New(),
		IncidentID: UIDownIncident.ID,
		Type:       "Telegram",
		PersonID:   &JohnDoe.ID,
		CreatedBy:  JohnDoe.ID,
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
