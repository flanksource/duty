package tests

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

type ConfigGenerator struct {
	PodPerDeployment, DeploymentPerNamespace, Namespaces, Nodes int
	NumChangesPerConfig                                         int
	NumInsightsPerConfig                                        int
	HealthyPercentage                                           int
	UnhealthyPercentage                                         int
	UnknownPercentage                                           int
	DeletedPercentage                                           int
	Tags                                                        map[string]string
	Generated                                                   struct {
		Configs       []models.ConfigItem
		Changes       []models.ConfigChange
		Analysis      []models.ConfigAnalysis
		Relationships []models.ConfigRelationship
	}
	count int
}

func (generator *ConfigGenerator) GenerateStatus() *string {
	randomNumber := rand.Intn(100)

	if randomNumber <= generator.UnhealthyPercentage {
		return lo.ToPtr("unhealthy")
	} else if randomNumber <= generator.UnhealthyPercentage+generator.UnknownPercentage {
		return lo.ToPtr("unknown")
	} else {
		return lo.ToPtr("healthy")
	}
}

func (generator *ConfigGenerator) GenerateDeleted() *time.Time {
	randomNumber := rand.Intn(100)

	if randomNumber <= generator.DeletedPercentage {
		return lo.ToPtr(time.Now().Add(-10 * 24 * time.Hour))
	}
	return nil
}

func (generator *ConfigGenerator) GenerateConfigItem(configType string, parent *models.ConfigItem) models.ConfigItem {
	changes := []models.ConfigChange{}
	analysis := []models.ConfigAnalysis{}
	generator.count++
	name := fmt.Sprintf("%s-%d", strings.Split(configType, "::")[1], generator.count)

	item := models.ConfigItem{
		ID:        uuid.New(),
		DeletedAt: generator.GenerateDeleted(),
		Type:      lo.ToPtr(configType),
		Name:      lo.ToPtr(name),
		Status:    generator.GenerateStatus(),
		Tags:      generator.Tags,
	}
	if parent != nil {
		item.ParentID = &parent.ID
	}

	for i := 1; i <= generator.NumChangesPerConfig; i++ {
		changes = append(changes, models.ConfigChange{
			ID:         uuid.New().String(),
			ConfigID:   item.ID.String(),
			ChangeType: "UPDATE",
			CreatedAt:  lo.ToPtr(time.Now().Add(-time.Duration(rand.Intn(60*72)) * time.Minute)),
			Summary:    "Change " + strconv.Itoa(i) + " for " + *item.Name,
			Source:     "test-generator",
		})
	}

	for i := 1; i <= generator.NumInsightsPerConfig; i++ {
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

	var nodes []models.ConfigItem

	cluster := generator.GenerateConfigItem("Kubernetes::Cluster", nil)

	for i := 0; i < generator.Nodes; i++ {
		nodes = append(nodes, generator.GenerateConfigItem("Kubernetes::Node", &cluster))
	}

	for i := 0; i < generator.Namespaces; i++ {
		ns := generator.GenerateConfigItem("Kubernetes::Namespace", &cluster)

		for j := 0; j < generator.DeploymentPerNamespace; j++ {
			deploy := generator.GenerateConfigItem("Kubernetes::Deployment", &ns)

			for k := 0; k < generator.PodPerDeployment; k++ {
				pod := generator.GenerateConfigItem("Kubernetes::Pod", &deploy)
				generator.Link(nodes[rand.Intn(len(nodes))], pod)
			}
		}
	}
}

func (generator *ConfigGenerator) Save(db *gorm.DB) error {
	tx := db.Begin()

	tx.CreateInBatches(generator.Generated.Configs, 100)
	tx.CreateInBatches(generator.Generated.Relationships, 100)
	tx.CreateInBatches(generator.Generated.Changes, 100)
	tx.CreateInBatches(generator.Generated.Analysis, 100)

	return tx.Commit().Error

}
