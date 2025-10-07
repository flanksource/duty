package tests

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

type testCase struct {
	rlsPayload    rls.Payload
	expectedCount *int64
}

func verifyConfigCount(tx *gorm.DB, rlsPayload rls.Payload, expectedCount int64) {
	Expect(rlsPayload.SetPostgresSessionRLS(tx)).To(BeNil())

	var count int64
	Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
	Expect(count).To(Equal(expectedCount))
}

var _ = Describe("RLS test", Ordered, func() {
	BeforeAll(func() {
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			Skip("RLS tests are disabled because DUTY_DB_DISABLE_RLS is set to true")
		}
	})

	var _ = Describe("views query", func() {
		var (
			tx           *gorm.DB
			totalConfigs int64
			awsConfigs   int64
		)

		BeforeAll(func() {
			Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws'").Model(&models.ConfigItem{}).Count(&awsConfigs).Error).To(BeNil())

			Expect(totalConfigs).To(Not(Equal(awsConfigs)))

			sqldb, err := DefaultContext.DB().DB()
			Expect(err).To(BeNil())

			// The migration_dependency_test can mess with the migration_logs so we clean and run migrations again
			Expect(DefaultContext.DB().Exec("DELETE FROM migration_logs").Error).To(BeNil())

			connString := DefaultContext.Value("db_url").(string)
			err = migrate.RunMigrations(sqldb, api.Config{ConnectionString: connString, EnableRLS: true})
			Expect(err).To(BeNil())

			tx = DefaultContext.DB().Begin()

			Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(BeNil())

			payload := rls.Payload{
				Config: []rls.Scope{
					{Tags: map[string]string{"cluster": "aws"}},
				},
			}
			Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

			err = job.RefreshConfigItemSummary7d(DefaultContext)
			Expect(err).To(BeNil())
		})

		AfterAll(func() {
			payload := rls.Payload{
				Config: []rls.Scope{
					{Tags: map[string]string{"cluster": "aws"}},
				},
			}
			Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())
			Expect(tx.Commit().Error).To(BeNil())
		})

		It("should call configs", func() {
			var count int64
			err := tx.Raw("SELECT COUNT(*) FROM configs").Scan(&count).Error
			Expect(err).To(BeNil())

			Expect(count).To(Equal(awsConfigs))
		})

		It("should call config_detail", func() {
			var count int64
			err := tx.Raw("SELECT COUNT(*) FROM config_detail").Scan(&count).Error
			Expect(err).To(BeNil())

			Expect(count).To(Equal(awsConfigs))
		})

		It("should call config_item_summary_7d", func() {
			var count int64
			err := tx.Raw("SELECT COUNT(*) FROM config_item_summary_7d").Scan(&count).Error
			Expect(err).To(BeNil())

			Expect(count).To(Equal(totalConfigs))
		})
	})

	var _ = Describe("config_items query", func() {
		var (
			tx                           *gorm.DB
			totalConfigs                 int64
			numConfigsWithAgent          int64
			numConfigsWithFlanksourceTag int64
			awsAndDemoCluster            int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'account' = 'flanksource'").Model(&models.ConfigItem{}).Count(&numConfigsWithFlanksourceTag).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ?", uuid.Nil).Model(&models.ConfigItem{}).Count(&numConfigsWithAgent).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws' OR tags->>'cluster' = 'demo'").Model(&models.ConfigItem{}).Count(&awsAndDemoCluster).Error).To(BeNil())
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())

					var currentRole string
					Expect(tx.Raw("SELECT CURRENT_USER").Scan(&currentRole).Error).To(BeNil())
					Expect(currentRole).To(Equal(role))
				})

				DescribeTable("JWT claim tests",
					func(tc testCase) {
						verifyConfigCount(tx, tc.rlsPayload, *tc.expectedCount)
					},
					Entry("no permissions", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags:   map[string]string{"cluster": "testing-cluster"},
									Agents: []string{"10000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("correct agent", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Agents: []string{"00000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("correct tag", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags: map[string]string{"account": "flanksource"},
								},
							},
						},
						expectedCount: &numConfigsWithFlanksourceTag,
					}),
					Entry("multiple tags (OR logic between scopes)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "aws"}},
								{Tags: map[string]string{"cluster": "demo"}},
							},
						},
						expectedCount: &awsAndDemoCluster,
					}),
					Entry("specific name", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{*dummy.EKSCluster.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wildcard name (match all)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalConfigs,
					}),
				)
			})
		}
	})
})
