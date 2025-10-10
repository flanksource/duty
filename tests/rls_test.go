package tests

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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

var _ = Describe("RLS test", Ordered, ContinueOnFailure, func() {
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
			awsConfigs                   int64
			awsAndDemoCluster            int64
			awsTagAndNilAgent            int64
			awsTagAndEKSName             int64
			awsAndFlanksourceTags        int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'account' = 'flanksource'").Model(&models.ConfigItem{}).Count(&numConfigsWithFlanksourceTag).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ?", uuid.Nil).Model(&models.ConfigItem{}).Count(&numConfigsWithAgent).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws'").Model(&models.ConfigItem{}).Count(&awsConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws' OR tags->>'cluster' = 'demo'").Model(&models.ConfigItem{}).Count(&awsAndDemoCluster).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws' AND agent_id = ?", uuid.Nil).Model(&models.ConfigItem{}).Count(&awsTagAndNilAgent).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws' AND name = ?", *dummy.EKSCluster.Name).Model(&models.ConfigItem{}).Count(&awsTagAndEKSName).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws' AND tags->>'account' = 'flanksource'").Model(&models.ConfigItem{}).Count(&awsAndFlanksourceTags).Error).To(BeNil())
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

				It("should allow access to all records when RLS is disabled", func() {
					payload := rls.Payload{
						Disable: true,
					}
					verifyConfigCount(tx, payload, totalConfigs)
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
					Entry("wildcard agent (match all)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Agents: []string{"*"}},
							},
						},
						expectedCount: &totalConfigs,
					}),
					Entry("tags AND agents (within scope)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags:   map[string]string{"cluster": "aws"},
									Agents: []string{uuid.Nil.String()},
								},
							},
						},
						expectedCount: &awsTagAndNilAgent,
					}),
					Entry("tags AND names (within scope)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags:  map[string]string{"cluster": "aws"},
									Names: []string{*dummy.EKSCluster.Name},
								},
							},
						},
						expectedCount: &awsTagAndEKSName,
					}),
					Entry("empty payload (no scopes)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple names (OR within names array)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{*dummy.EKSCluster.Name, "non-existent-config"}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("mixed scope criteria (OR logic between scopes)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "aws"}},
								{Agents: []string{uuid.Nil.String()}},
								{Names: []string{*dummy.EKSCluster.Name}},
							},
						},
						expectedCount: &numConfigsWithAgent, // Should be union of all three scopes
					}),
					Entry("invalid agent UUID (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Agents: []string{"not-a-valid-uuid"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in agents array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Agents: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty tag value (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": ""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{strings.ToUpper(*dummy.EKSCluster.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase tag value (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "AWS"}}, // uppercase
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("duplicate scopes (should work same as single)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "aws"}},
								{Tags: map[string]string{"cluster": "aws"}}, // duplicate
							},
						},
						expectedCount: &awsConfigs, // Should be same as single scope
					}),
					Entry("conflicting criteria within scope (agent matches but name doesn't)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},          // matches many
									Names:  []string{"non-existent-config-name"}, // matches none
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // AND logic means both must match
					}),
					Entry("special characters in name (unicode)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{"config-名前-🚀"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple agents in single scope (OR within agents array)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Agents: []string{
										uuid.Nil.String(),
										"10000000-0000-0000-0000-000000000000",
									},
								},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("multiple tags in single scope (AND logic)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags: map[string]string{
										"cluster": "aws",
										"account": "flanksource",
									},
								},
							},
						},
						expectedCount: &awsAndFlanksourceTags,
					}),
					Entry("mixed valid and invalid agents", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Agents: []string{
										"not-a-uuid",
										uuid.Nil.String(),
										"also-invalid",
									},
								},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("very long agent list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Agents: append(
										[]string{uuid.Nil.String()},
										func() []string {
											agents := make([]string, 99)
											for i := range agents {
												agents[i] = uuid.New().String()
											}
											return agents
										}()...,
									),
								},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("very long names list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Names: append(
										[]string{*dummy.EKSCluster.Name},
										func() []string {
											names := make([]string, 99)
											for i := range names {
												names[i] = fmt.Sprintf("non-existent-config-%d", i)
											}
											return names
										}()...,
									),
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("very many scopes (stress test)", testCase{
						rlsPayload: rls.Payload{
							Config: append(
								[]rls.Scope{{Tags: map[string]string{"cluster": "aws"}}},
								func() []rls.Scope {
									scopes := make([]rls.Scope, 49)
									for i := range scopes {
										scopes[i] = rls.Scope{
											Tags: map[string]string{"cluster": fmt.Sprintf("non-existent-%d", i)},
										}
									}
									return scopes
								}()...,
							),
						},
						expectedCount: &awsConfigs,
					}),
					Entry("tag with special characters in key", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster-name-with-dashes": "value"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("tag key exists but value doesn't match", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "non-existent-value"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple tags where only one matches", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags: map[string]string{
										"cluster":     "aws",
										"nonexistent": "should-fail",
									},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty tag map in scope", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags:   map[string]string{},
									Agents: []string{uuid.Nil.String()},
								},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("whitespace-only values", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Names: []string{"   "},
									Tags:  map[string]string{"cluster": "   "},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long name string", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{strings.Repeat("a", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long tag value", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": strings.Repeat("x", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard in middle", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{"Production*EKS"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard prefix", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Names: []string{"*EKS"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with overlapping results", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "aws"}},
								{Names: []string{*dummy.EKSCluster.Name}},
							},
						},
						expectedCount: &awsConfigs,
					}),
					Entry("agent UUID with uppercase", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Agents: []string{strings.ToUpper(uuid.Nil.String())}},
							},
						},
						expectedCount: &numConfigsWithAgent,
					}),
					Entry("newline in tag value", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{Tags: map[string]string{"cluster": "aws\nmalicious"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("valid tag + valid agent + invalid name (AND within scope)", testCase{
						rlsPayload: rls.Payload{
							Config: []rls.Scope{
								{
									Tags:   map[string]string{"cluster": "aws"},
									Agents: []string{uuid.Nil.String()},
									Names:  []string{"non-existent"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
				)
			})
		}
	})

	var _ = Describe("components query", func() {
		var (
			tx                     *gorm.DB
			totalComponents        int64
			numComponentsWithAgent int64
			agentAndLogisticsName  int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Component{}).Count(&totalComponents).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ?", uuid.Nil).Model(&models.Component{}).Count(&numComponentsWithAgent).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ? AND name = ?", uuid.Nil, dummy.Logistics.Name).Model(&models.Component{}).Count(&agentAndLogisticsName).Error).To(BeNil())
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
						Expect(tc.rlsPayload.SetPostgresSessionRLS(tx)).To(BeNil())

						var count int64
						Expect(tx.Model(&models.Component{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no permissions", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{"10000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("correct agent", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
								},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("specific name", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{dummy.Logistics.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wildcard name (match all)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalComponents,
					}),
					Entry("agents AND names (within scope)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{dummy.Logistics.Name},
								},
							},
						},
						expectedCount: &agentAndLogisticsName,
					}),
					Entry("empty payload (no scopes)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple names (OR within names array)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{dummy.Logistics.Name, "non-existent-component"}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("mixed scope criteria (OR logic between scopes)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Agents: []string{uuid.Nil.String()}},
								{Names: []string{dummy.Logistics.Name}},
							},
						},
						expectedCount: &numComponentsWithAgent, // Should be union of both scopes
					}),
					Entry("invalid agent UUID (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Agents: []string{"invalid-uuid"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in agents array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Agents: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{strings.ToUpper(dummy.Logistics.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("conflicting criteria within scope (agent matches but name doesn't)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{"non-existent-component"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // AND logic means both must match
					}),
					Entry("multiple agents in single scope", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{
										uuid.Nil.String(),
										"10000000-0000-0000-0000-000000000000",
									},
								},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("mixed valid and invalid agents", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{
										"not-a-uuid",
										uuid.Nil.String(),
									},
								},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("very long agent list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: append(
										[]string{uuid.Nil.String()},
										func() []string {
											agents := make([]string, 99)
											for i := range agents {
												agents[i] = uuid.New().String()
											}
											return agents
										}()...,
									),
								},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("very long names list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Names: append(
										[]string{dummy.Logistics.Name},
										func() []string {
											names := make([]string, 99)
											for i := range names {
												names[i] = fmt.Sprintf("non-existent-component-%d", i)
											}
											return names
										}()...,
									),
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("whitespace-only name", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{"   "}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long name string", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{strings.Repeat("a", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard in middle", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Names: []string{"Log*tics"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with overlapping results", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Agents: []string{uuid.Nil.String()}},
								{Names: []string{dummy.Logistics.Name}},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("agent UUID with uppercase", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{Agents: []string{strings.ToUpper(uuid.Nil.String())}},
							},
						},
						expectedCount: &numComponentsWithAgent,
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("valid agent + invalid name (AND within scope)", testCase{
						rlsPayload: rls.Payload{
							Component: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{"non-existent"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
				)
			})
		}
	})

	var _ = Describe("playbooks query", func() {
		var (
			tx             *gorm.DB
			totalPlaybooks int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Playbook{}).Count(&totalPlaybooks).Error).To(BeNil())
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
						Expect(tc.rlsPayload.SetPostgresSessionRLS(tx)).To(BeNil())

						var count int64
						Expect(tx.Model(&models.Playbook{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no permissions", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Names: []string{"non-existent-playbook"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("specific name", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wildcard name (match all)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalPlaybooks,
					}),
					Entry("empty payload (no scopes)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple names (OR within names array)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name, "non-existent-playbook"}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{strings.ToUpper(dummy.EchoConfig.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("duplicate scopes (should work same as single)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name}},
								{Names: []string{dummy.EchoConfig.Name}}, // duplicate
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("very long names list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Names: append(
										[]string{dummy.EchoConfig.Name},
										func() []string {
											names := make([]string, 99)
											for i := range names {
												names[i] = fmt.Sprintf("non-existent-playbook-%d", i)
											}
											return names
										}()...,
									),
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("whitespace-only name", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"   "}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long name string", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{strings.Repeat("a", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard in middle", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"Echo*Config"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard prefix", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"*Config"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with overlapping results", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name}},
								{Names: []string{dummy.EchoConfig.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("agents defined in scope (should be ignored for playbooks)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Agents: []string{"10000000-0000-0000-0000-000000000000"},
									Names:  []string{dummy.EchoConfig.Name},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)), // Should match because agents should be ignored
					}),
					Entry("tags only in scope (should deny access - no applicable fields)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Tags: map[string]string{"cluster": "homelab", "namespace": "default"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // Should deny because playbooks don't support tags
					}),
					Entry("tags and agents only in scope (should deny access - no applicable fields)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Tags:   map[string]string{"cluster": "aws"},
									Agents: []string{"10000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // Should deny because playbooks support neither tags nor agents
					}),
					Entry("specific ID (should grant access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{ID: dummy.EchoConfig.ID.String()},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wrong ID (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{ID: "00000000-0000-0000-0000-000000000000"},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("ID + matching name (AND logic - should grant access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									ID:    dummy.EchoConfig.ID.String(),
									Names: []string{dummy.EchoConfig.Name},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("ID + non-matching name (AND logic - should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									ID:    dummy.EchoConfig.ID.String(),
									Names: []string{"wrong-name"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with different IDs (OR logic)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{ID: dummy.EchoConfig.ID.String()},
								{ID: dummy.RestartPod.ID.String()},
							},
						},
						expectedCount: lo.ToPtr(int64(2)),
					}),
				)
			})
		}
	})

	var _ = Describe("canaries query", func() {
		var (
			tx                   *gorm.DB
			totalCanaries        int64
			numCanariesWithAgent int64
			agentAndCanaryName   int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Canary{}).Count(&totalCanaries).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ?", uuid.Nil).Model(&models.Canary{}).Count(&numCanariesWithAgent).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ? AND name = ?", uuid.Nil, dummy.LogisticsAPICanary.Name).Model(&models.Canary{}).Count(&agentAndCanaryName).Error).To(BeNil())
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
						Expect(tc.rlsPayload.SetPostgresSessionRLS(tx)).To(BeNil())

						var count int64
						Expect(tx.Model(&models.Canary{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no permissions", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{"10000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("correct agent", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
								},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("specific name", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{dummy.LogisticsAPICanary.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wildcard name (match all)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalCanaries,
					}),
					Entry("agents AND names (within scope)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{dummy.LogisticsAPICanary.Name},
								},
							},
						},
						expectedCount: &agentAndCanaryName,
					}),
					Entry("empty payload (no scopes)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple names (OR within names array)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{dummy.LogisticsAPICanary.Name, "non-existent-canary"}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("mixed scope criteria (OR logic between scopes)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Agents: []string{uuid.Nil.String()}},
								{Names: []string{dummy.LogisticsAPICanary.Name}},
							},
						},
						expectedCount: &numCanariesWithAgent, // Should be union of both scopes
					}),
					Entry("invalid agent UUID (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Agents: []string{"not-valid-uuid"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in agents array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Agents: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{strings.ToUpper(dummy.LogisticsAPICanary.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("conflicting criteria within scope (agent matches but name doesn't)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{"non-existent-canary"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // AND logic means both must match
					}),
					Entry("multiple agents in single scope", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{
										uuid.Nil.String(),
										"10000000-0000-0000-0000-000000000000",
									},
								},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("mixed valid and invalid agents", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{
										"not-a-uuid",
										uuid.Nil.String(),
									},
								},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("very long agent list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: append(
										[]string{uuid.Nil.String()},
										func() []string {
											agents := make([]string, 99)
											for i := range agents {
												agents[i] = uuid.New().String()
											}
											return agents
										}()...,
									),
								},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("very long names list (stress test)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Names: append(
										[]string{dummy.LogisticsAPICanary.Name},
										func() []string {
											names := make([]string, 99)
											for i := range names {
												names[i] = fmt.Sprintf("non-existent-canary-%d", i)
											}
											return names
										}()...,
									),
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("whitespace-only name", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{"   "}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long name string", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{strings.Repeat("a", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard in middle", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{"Logistics*Canary"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard prefix", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Names: []string{"*Canary"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with overlapping results", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Agents: []string{uuid.Nil.String()}},
								{Names: []string{dummy.LogisticsAPICanary.Name}},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("agent UUID with uppercase", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{Agents: []string{strings.ToUpper(uuid.Nil.String())}},
							},
						},
						expectedCount: &numCanariesWithAgent,
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("valid agent + invalid name (AND within scope)", testCase{
						rlsPayload: rls.Payload{
							Canary: []rls.Scope{
								{
									Agents: []string{uuid.Nil.String()},
									Names:  []string{"non-existent"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
				)
			})
		}
	})

	var _ = Describe("playbook_runs query", func() {
		var (
			tx                  *gorm.DB
			totalPlaybookRuns   int64
			echoConfigRunsCount int64
			restartPodRunsCount int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.PlaybookRun{}).Count(&totalPlaybookRuns).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("playbook_id = ?", dummy.EchoConfig.ID).Model(&models.PlaybookRun{}).Count(&echoConfigRunsCount).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("playbook_id = ?", dummy.RestartPod.ID).Model(&models.PlaybookRun{}).Count(&restartPodRunsCount).Error).To(BeNil())

			Expect(totalPlaybookRuns).To(BeNumerically(">", 0), "No playbook runs found in test data")
			Expect(echoConfigRunsCount).To(BeNumerically(">", 0), "No playbook runs found for EchoConfig playbook")
			Expect(restartPodRunsCount).To(BeNumerically(">", 0), "No playbook runs found for RestartPod playbook")
			Expect(totalPlaybookRuns).To(Equal(echoConfigRunsCount + restartPodRunsCount))
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
						Expect(tc.rlsPayload.SetPostgresSessionRLS(tx)).To(BeNil())

						var count int64
						Expect(tx.Model(&models.PlaybookRun{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no permissions (empty scopes array)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("no permissions (non-existent playbook)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{
									Names: []string{"non-existent-playbook"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("access only echo-config playbook runs", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name}},
							},
						},
						expectedCount: &echoConfigRunsCount,
					}),
					Entry("access only restart-pod playbook runs", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.RestartPod.Name}},
							},
						},
						expectedCount: &restartPodRunsCount,
					}),
					Entry("access both playbooks (OR logic)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{dummy.EchoConfig.Name, dummy.RestartPod.Name}},
							},
						},
						expectedCount: &totalPlaybookRuns,
					}),
					Entry("wildcard playbook name (match all runs)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalPlaybookRuns,
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase playbook name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{strings.ToUpper(dummy.EchoConfig.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("whitespace-only name", testCase{
						rlsPayload: rls.Payload{
							Playbook: []rls.Scope{
								{Names: []string{"   "}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
				)
			})
		}
	})
})
