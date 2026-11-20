package tests

import (
	"fmt"
	"strings"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/upstream"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("Upstream deletion errors", func() {
	var component models.Component
	var config models.ConfigItem
	var topology models.Topology
	var failureFunction string

	ginkgo.BeforeEach(func() {
		db := DefaultContext.DB()
		component = models.Component{ID: uuid.New(), Name: "upstream-delete-component"}
		config = models.ConfigItem{ID: uuid.New(), Name: lo.ToPtr("upstream-delete-config"), Type: lo.ToPtr("test"), ConfigClass: "test"}
		topology = models.Topology{ID: uuid.New(), Name: "upstream-delete-topology", Source: models.SourcePush}
		Expect(db.Create(&component).Error).NotTo(HaveOccurred())
		ginkgo.DeferCleanup(func() { Expect(db.Delete(&component).Error).NotTo(HaveOccurred()) })
		Expect(db.Create(&config).Error).NotTo(HaveOccurred())
		ginkgo.DeferCleanup(func() { Expect(db.Delete(&config).Error).NotTo(HaveOccurred()) })
		Expect(db.Create(&topology).Error).NotTo(HaveOccurred())
		ginkgo.DeferCleanup(func() { Expect(db.Delete(&topology).Error).NotTo(HaveOccurred()) })
		failureFunction = fmt.Sprintf("upstream_delete_failure_%x", uuid.New().ID())
		Expect(db.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'blocked deletion on %%', TG_TABLE_NAME; END $$`, failureFunction)).Error).NotTo(HaveOccurred())
		ginkgo.DeferCleanup(func() {
			Expect(db.Exec("DROP FUNCTION " + failureFunction + "() CASCADE").Error).NotTo(HaveOccurred())
		})
	})

	ginkgo.DescribeTable("returns every deletion failure while attempting the remaining resources", func(failingTables []string) {
		db := DefaultContext.DB()
		ids := map[string]uuid.UUID{"components": component.ID, "config_items": config.ID, "topologies": topology.ID}
		for _, table := range failingTables {
			Expect(db.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE DELETE ON %s
				FOR EACH ROW WHEN (OLD.id = '%s') EXECUTE FUNCTION %s()`, failureFunction, table, ids[table], failureFunction)).Error).NotTo(HaveOccurred())
		}
		err := upstream.DeleteOnUpstream(DefaultContext, uuid.Nil, &upstream.PushData{
			Components: []models.Component{component}, ConfigItems: []models.ConfigItem{config}, Topologies: []models.Topology{topology},
		})
		if len(failingTables) == 0 {
			Expect(err).NotTo(HaveOccurred())
		} else {
			for _, table := range failingTables {
				Expect(err).To(MatchError(ContainSubstring("error deleting " + strings.ReplaceAll(table, "_", " "))))
			}
		}
		for table, id := range ids {
			var remaining int64
			Expect(db.Table(table).Where("id = ?", id).Count(&remaining).Error).NotTo(HaveOccurred())
			Expect(remaining).To(Equal(lo.Ternary(lo.Contains(failingTables, table), int64(1), int64(0))), table)
		}
	},
		ginkgo.Entry("successful deletes", []string{}),
		ginkgo.Entry("component failure", []string{"components"}),
		ginkgo.Entry("config failure", []string{"config_items"}),
		ginkgo.Entry("component and config failures", []string{"components", "config_items"}),
		ginkgo.Entry("a later failure retains earlier failures", []string{"components", "config_items", "topologies"}),
	)
})
