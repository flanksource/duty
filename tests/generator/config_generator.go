package generator

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type ConfigTypeHealthRequirements struct {
	HealthyPercentage   int
	UnhealthyPercentage int
	WarningPercentage   int
	UnknownPercentage   int
}

func (t *ConfigTypeHealthRequirements) SetDefaults() {
	if t.IsEmpty() {
		t.HealthyPercentage = 100
	}
}

func (t *ConfigTypeHealthRequirements) IsEmpty() bool {
	return t.HealthyPercentage+t.UnhealthyPercentage+t.WarningPercentage+t.UnknownPercentage == 0
}

func (t *ConfigTypeHealthRequirements) IsValid() bool {
	return t.HealthyPercentage+t.UnhealthyPercentage+t.WarningPercentage+t.UnknownPercentage == 100
}

type ConfigTypeRequirements struct {
	Status ConfigTypeHealthRequirements

	Count   int
	Deleted int

	NumChangesPerConfig  int
	NumInsightsPerConfig int
}

type Generated struct {
	Configs       []models.ConfigItem
	Changes       []models.ConfigChange
	Analysis      []models.ConfigAnalysis
	Relationships []models.ConfigRelationship
}

func (t *Generated) ConfigByTypes(configType ...string) []models.ConfigItem {
	output := make([]models.ConfigItem, 0)
	for _, config := range t.Configs {
		if lo.Contains(configType, *config.Type) && config.DeletedAt == nil {
			output = append(output, config)
		}
	}

	return output
}

func (t *Generated) Total() int {
	return len(t.Configs) + len(t.Changes) + len(t.Analysis) + len(t.Relationships)
}

type ConfigGenerator struct {
	Namespaces, Nodes                                                  ConfigTypeRequirements
	PodsPerReplicaSet, ReplicaSetPerDeployment, DeploymentPerNamespace ConfigTypeRequirements
	Tags                                                               map[string]string
	Generated                                                          Generated

	count int
}

func (generator *ConfigGenerator) GenerateConfigItem(configType, status string, deletedAt *time.Time, parent *models.ConfigItem, req ConfigTypeRequirements) models.ConfigItem {
	changes := []models.ConfigChange{}
	analysis := []models.ConfigAnalysis{}
	generator.count++
	name := fmt.Sprintf("%s-%d", strings.Split(configType, "::")[1], generator.count)

	item := models.ConfigItem{
		ID:        uuid.New(),
		DeletedAt: deletedAt,
		Type:      lo.ToPtr(configType),
		Name:      lo.ToPtr(name),
		Status:    &status,
		Health:    lo.ToPtr(models.Health(status)),
		Tags:      generator.Tags,
	}
	if parent != nil {
		item.ParentID = &parent.ID
	}

	for i := 1; i <= req.NumChangesPerConfig; i++ {
		changes = append(changes, models.ConfigChange{
			ID:         uuid.New().String(),
			ConfigID:   item.ID.String(),
			ChangeType: "UPDATE",
			CreatedAt:  lo.ToPtr(time.Now().Add(-time.Duration(rand.Intn(60*72)) * time.Minute)),
			Summary:    "Change " + strconv.Itoa(i) + " for " + *item.Name,
			Source:     "test-generator",
		})
	}

	for i := 1; i <= req.NumInsightsPerConfig; i++ {
		analysis = append(analysis, models.ConfigAnalysis{
			ID:           uuid.New(),
			ConfigID:     item.ID,
			AnalysisType: models.AnalysisTypeAvailability,
			LastObserved: lo.ToPtr(time.Now().Add(-time.Duration(rand.Intn(60)) * time.Minute)),
			Summary:      "Insight " + strconv.Itoa(i) + " for " + *item.Name,
			Source:       "test-generator",
		})
	}

	generator.Generated.Configs = append(generator.Generated.Configs, item)
	generator.Generated.Changes = append(generator.Generated.Changes, changes...)
	generator.Generated.Analysis = append(generator.Generated.Analysis, analysis...)

	return item
}

func (generator *ConfigGenerator) Link(parent, child models.ConfigItem) {
	link := models.ConfigRelationship{
		ConfigID:  parent.ID.String(),
		RelatedID: child.ID.String(),
	}
	generator.Generated.Relationships = append(generator.Generated.Relationships, link)
}

func (generator *ConfigGenerator) GenerateKubernetes() {
	cluster := generator.GenerateConfigItem("Kubernetes::Cluster", "unknown", nil, nil, ConfigTypeRequirements{})
	nodes := generator.generateNodes(&cluster)

	for i := 0; i < generator.Namespaces.Count; i++ {
		ns := generator.GenerateConfigItem("Kubernetes::Namespace", "Healthy", deletedTime(i, generator.Namespaces.Deleted), &cluster, generator.Namespaces)
		generator.generateDeployments(ns, nodes)
	}
}

func (generator *ConfigGenerator) generateNodes(cluster *models.ConfigItem) []models.ConfigItem {
	var nodes []models.ConfigItem
	nodeStatuses := genStatuses(generator.Nodes.Count, generator.Nodes.Status)
	for i := 0; i < generator.Nodes.Count; i++ {
		node := generator.GenerateConfigItem("Kubernetes::Node", nodeStatuses[i], deletedTime(i, generator.Nodes.Deleted), cluster, generator.Nodes)
		nodes = append(nodes, node)
	}
	return nodes
}

func (generator *ConfigGenerator) generateDeployments(ns models.ConfigItem, nodes []models.ConfigItem) {
	deploymentStatuses := genStatuses(generator.DeploymentPerNamespace.Count, generator.DeploymentPerNamespace.Status)
	for j := 0; j < generator.DeploymentPerNamespace.Count; j++ {
		deletedAt := getDeletedAt(ns.DeletedAt, j, generator.DeploymentPerNamespace.Deleted)
		deploy := generator.GenerateConfigItem("Kubernetes::Deployment", deploymentStatuses[j], deletedAt, &ns, generator.DeploymentPerNamespace)
		generator.generateReplicaSets(deploy, nodes)
	}
}

func (generator *ConfigGenerator) generateReplicaSets(deploy models.ConfigItem, nodes []models.ConfigItem) {
	replicaSetStatuses := genStatuses(generator.ReplicaSetPerDeployment.Count, generator.ReplicaSetPerDeployment.Status)
	for k := 0; k < generator.ReplicaSetPerDeployment.Count; k++ {
		deletedAt := getDeletedAt(deploy.DeletedAt, k, generator.ReplicaSetPerDeployment.Deleted)
		replicaSet := generator.GenerateConfigItem("Kubernetes::ReplicaSet", replicaSetStatuses[k], deletedAt, &deploy, generator.ReplicaSetPerDeployment)
		generator.generatePods(replicaSet, deploy, nodes)
	}
}

func (generator *ConfigGenerator) generatePods(replicaSet, deploy models.ConfigItem, nodes []models.ConfigItem) {
	podStatuses := genStatuses(generator.PodsPerReplicaSet.Count, generator.PodsPerReplicaSet.Status)
	for l := 0; l < generator.PodsPerReplicaSet.Count; l++ {
		deletedAt := getDeletedAt(replicaSet.DeletedAt, l, generator.PodsPerReplicaSet.Deleted)
		pod := generator.GenerateConfigItem("Kubernetes::Pod", podStatuses[l], deletedAt, &replicaSet, generator.PodsPerReplicaSet)
		generator.Link(deploy, pod)
		generator.Link(nodes[rand.Intn(len(nodes))], pod)
	}
}

func getDeletedAt(parentDeletedAt *time.Time, currentIndex, deletedNeeded int) *time.Time {
	if parentDeletedAt != nil {
		return parentDeletedAt
	}
	return deletedTime(currentIndex, deletedNeeded)
}

func (generator *ConfigGenerator) Save(db *gorm.DB) error {
	tx := db.Begin()

	tx.CreateInBatches(generator.Generated.Configs, 100)
	tx.CreateInBatches(generator.Generated.Relationships, 100)
	tx.CreateInBatches(generator.Generated.Changes, 100)
	tx.CreateInBatches(generator.Generated.Analysis, 100)

	return tx.Commit().Error
}

func (generator *ConfigGenerator) Destroy(db *gorm.DB) error {
	tx := db.Begin()

	tx.Delete(&generator.Generated.Analysis)
	tx.Delete(&generator.Generated.Changes)
	tx.Delete(&generator.Generated.Relationships)
	tx.Delete(&generator.Generated.Configs)
	return tx.Commit().Error
}

func deletedTime(currentIndex int, deletedNeeded int) *time.Time {
	if currentIndex < deletedNeeded {
		return lo.ToPtr(time.Now().Add(-time.Duration(rand.Intn(60*24)) * time.Minute))
	}

	return nil
}

// genStatuses generates a slice of status strings based on the provided ConfigTypeHealthRequirements.
func genStatuses(total int, req ConfigTypeHealthRequirements) []string {
	req.SetDefaults()

	output := make([]string, 0, total)
	healthyCount := int(float64(total) * float64(req.HealthyPercentage/100))
	unhealthyCount := int(float64(total) * float64(req.UnhealthyPercentage/100))
	warningCount := int(float64(total) * float64(req.WarningPercentage/100))
	unknownCount := int(float64(total) * float64(req.WarningPercentage/100))

	for i := 0; i < healthyCount; i++ {
		output = append(output, "healthy")
	}
	for i := 0; i < unhealthyCount; i++ {
		output = append(output, "unhealthy")
	}
	for i := 0; i < warningCount; i++ {
		output = append(output, "warning")
	}
	for i := 0; i < unknownCount; i++ {
		output = append(output, "unknown")
	}

	return output
}
