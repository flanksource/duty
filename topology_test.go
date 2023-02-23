package duty

import (
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/models"
	_ "github.com/flanksource/duty/types"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTopology(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
}

func generateDummyComponent(name string) models.Component {
	return models.Component{
		ID:         uuid.New(),
		Name:       gofakeit.AppName(),
		ExternalId: gofakeit.UUID(),
		Status:     models.ComponentStatusHealthy,
	}
}

var _ = ginkgo.Describe("Models creation", func() {
	/*
	   level0 - root
	   level1A,B,C - children of level0

	   level2D,E - children of level1A
	   level2F,G - children of level1B

	   level3H,J - children of level2D
	   level3K,M - children of level2G
	*/

	gorm, err := NewGorm(pgUrl, DefaultGormConfig())

	level0 := generateDummyComponent("level0")

	level0X := generateDummyComponent("level0X")

	level1A := generateDummyComponent("level1A")
	level1A.ParentId = &level0.ID

	level1B := generateDummyComponent("level1B")
	level1B.ParentId = &level0.ID

	level1C := generateDummyComponent("level1C")
	level1C.ParentId = &level0.ID

	level2D := generateDummyComponent("level2D")
	level2D.ParentId = &level1A.ID

	level2E := generateDummyComponent("level2E")
	level2E.ParentId = &level1A.ID

	level2F := generateDummyComponent("level2F")
	level2F.ParentId = &level1B.ID

	level2G := generateDummyComponent("level2G")
	level2G.Status = models.ComponentStatusUnhealthy
	level2G.ParentId = &level1B.ID

	level3H := generateDummyComponent("level3H")
	level3H.ParentId = &level2D.ID

	level3J := generateDummyComponent("level3J")
	level3J.ParentId = &level2D.ID

	level3K := generateDummyComponent("level3K")
	level2G.Status = models.ComponentStatusUnhealthy
	level3K.ParentId = &level2G.ID

	level3M := generateDummyComponent("level3M")
	level3M.ParentId = &level2G.ID

	// TODO: Generate dummy incidents and link with component via evidence

	// TODO: Generate dummy config analysis and link with components via config_relationship

	// TODO: Add a soft component relationship and test that parent should have soft child but soft child should not have parent

	allComponents := []models.Component{
		level0, level0X,
		level1A, level1B, level1C,
		level2D, level2E, level2F, level2G,
		level3H, level3J, level3K, level3M,
	}
	_ = allComponents
	ginkgo.It("should be able to create models", func() {
		logger.Infof("Running model create against %s", pgUrl)
		for _, c := range dummy.AllDummyPeople {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}

		for _, c := range dummy.AllDummyComponents {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}
		for _, c := range dummy.AllDummyConfigs {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}
		for _, c := range dummy.AllDummyConfigAnalysis {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}
		for _, c := range dummy.AllDummyConfigComponentRelationships {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}
		for _, c := range dummy.AllDummyIncidents {
			err = gorm.Create(&c).Error
			Expect(err).ToNot(HaveOccurred())
		}

	})

	ginkgo.It("able to fetch model", func() {
		var comp models.Component
		Expect(gorm.Table("components").First(&comp).Error).ToNot(HaveOccurred())
		Expect(comp.Status).To(Equal(models.ComponentStatus("healthy")))
	})

	ginkgo.It("Should create tree", func() {
		mytree, err := QueryTopology()
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("\n\n")
		for _, c := range mytree {
			fmt.Printf("- %s {analysis: %v}\n", c.Name, c.Summary)
			for _, cc := range c.Components {
				fmt.Printf("  |- %s {analysis: %v}\n", cc.Name, cc.Summary)
				for _, ccc := range cc.Components {
					fmt.Printf("    |- %s {analysis: %v}\n", ccc.Name, ccc.Summary)
					for _, cccc := range ccc.Components {
						fmt.Printf("      |- %s {analysis: %v}\n", cccc.Name, cccc.Summary)
					}
				}
			}
		}
		Expect(true).To(Equal(true))
	})
})
