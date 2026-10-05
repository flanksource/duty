package tests

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/job"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rbac/membership"
	"github.com/flanksource/duty/rbac/policy"
	"github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
)

type testCase struct {
	rlsPayload    rls.Payload
	expectedCount *int64
}

// grantCase is a claim and the number of rows it must list.
type grantCase struct {
	name     string
	payload  func() rls.Payload
	expected func() int64
}

// grants returns grants of the given sets of Scope ids.
func grants(sets ...[]string) *rls.Grants {
	g := rls.NoRows()
	for _, set := range sets {
		g.Add(set...)
	}
	return g
}

// grantRows stores the resources as members of a new Scope, inside the transaction, and returns grants naming
// that Scope. It must be called before the transaction switches role.
func grantRows(tx *gorm.DB, kind string, ids ...uuid.UUID) *rls.Grants {
	GinkgoHelper()
	scopeID := uuid.New()
	for _, id := range ids {
		Expect(tx.Exec("INSERT INTO scope_members (scope_id, resource_type, resource_id) VALUES (?, ?, ?)", scopeID, kind, id).Error).To(Succeed())
	}
	return grants([]string{scopeID.String()})
}

func countRows(tx *gorm.DB, table string, payload rls.Payload) int64 {
	GinkgoHelper()
	Expect(payload.SetPostgresSessionRLS(tx)).To(Succeed())

	var count int64
	Expect(tx.Table(table).Count(&count).Error).To(Succeed())
	return count
}

func countAll(table, where string, args ...any) int64 {
	GinkgoHelper()
	var count int64
	query := DefaultContext.DB().Table(table)
	if where != "" {
		query = query.Where(where, args...)
	}
	Expect(query.Count(&count).Error).To(Succeed())
	return count
}

var _ = Describe("RLS test", Ordered, ContinueOnFailure, func() {
	var (
		scopeIDs []uuid.UUID

		awsScope, demoScope, localAgentScope, eksScope, allConfigsScope string
		logisticsComponentScope, gcpComponentScope                      string
		echoPlaybookScope, mcPlaybooksScope                             string
		logisticsAPICanaryScope, gcpCanaryScope                         string
		apiHealthCheckScope                                             string
		unbuiltScope                                                    = uuid.NewString()
	)

	buildScope := func(name string, targets ...membership.Target) string {
		GinkgoHelper()
		scope := models.Scope{ID: uuid.New(), Name: "rls-" + name, Namespace: "rls-test", Targets: types.JSON(`[]`)}
		Expect(DefaultContext.DB().Create(&scope).Error).To(Succeed())
		_, err := membership.Rebuild(DefaultContext, scope.ID, scope.Targets, targets)
		Expect(err).ToNot(HaveOccurred())
		scopeIDs = append(scopeIDs, scope.ID)
		return scope.ID.String()
	}

	target := func(kind string, selector types.ResourceSelector) membership.Target {
		return membership.Target{Type: kind, Selector: selector}
	}

	BeforeAll(func() {
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			Skip("RLS tests are disabled because DUTY_DB_DISABLE_RLS is set to true")
		}

		awsScope = buildScope("aws", target(policy.ResourceConfig, types.ResourceSelector{TagSelector: "cluster=aws"}))
		demoScope = buildScope("demo", target(policy.ResourceConfig, types.ResourceSelector{TagSelector: "cluster=demo"}))
		localAgentScope = buildScope("local-agent", target(policy.ResourceConfig, types.ResourceSelector{Agent: uuid.Nil.String()}))
		eksScope = buildScope("eks", target(policy.ResourceConfig, types.ResourceSelector{Name: *dummy.EKSCluster.Name}))
		allConfigsScope = buildScope("all-configs", target(policy.ResourceConfig, types.ResourceSelector{Name: "*"}))
		logisticsComponentScope = buildScope("logistics-component", target(policy.ResourceComponent, types.ResourceSelector{Name: dummy.Logistics.Name}))
		gcpComponentScope = buildScope("gcp-components", target(policy.ResourceComponent, types.ResourceSelector{Agent: dummy.GCPAgent.ID.String()}))
		echoPlaybookScope = buildScope("echo-playbook", target(policy.ResourcePlaybook, types.ResourceSelector{Name: dummy.EchoConfig.Name}))
		mcPlaybooksScope = buildScope("mc-playbooks", target(policy.ResourcePlaybook, types.ResourceSelector{Namespace: dummy.EchoConfig.Namespace}))
		logisticsAPICanaryScope = buildScope("logistics-api-canary", target(policy.ResourceCanary, types.ResourceSelector{Name: dummy.LogisticsAPICanary.Name}))
		gcpCanaryScope = buildScope("gcp-canaries", target(policy.ResourceCanary, types.ResourceSelector{Agent: dummy.GCPAgent.ID.String()}))
		apiHealthCheckScope = buildScope("api-health-check", target(policy.ResourceCheck, types.ResourceSelector{ID: dummy.LogisticsAPIHealthHTTPCheck.ID.String()}))
	})

	AfterAll(func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM scopes WHERE id IN ?", scopeIDs).Error).To(Succeed())
	})

	var _ = Describe("views query", func() {
		var tx *gorm.DB

		BeforeAll(func() {
			sqldb, err := DefaultContext.DB().DB()
			Expect(err).To(BeNil())

			// The migration_dependency_test can mess with the migration_logs so we clean and run migrations again
			Expect(DefaultContext.DB().Exec("DELETE FROM migration_logs").Error).To(BeNil())

			connString := DefaultContext.Value("db_url").(string)
			err = migrate.RunMigrations(sqldb, api.Config{ConnectionString: connString, EnableRLS: true})
			Expect(err).To(BeNil())

			tx = DefaultContext.DB().Begin()

			Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(BeNil())
			Expect((rls.Payload{Config: grants([]string{awsScope})}).SetPostgresSessionRLS(tx)).To(BeNil())

			err = job.RefreshConfigItemSummary7d(DefaultContext)
			Expect(err).To(BeNil())
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		It("should call configs", func() {
			var count int64
			Expect(tx.Raw("SELECT COUNT(*) FROM configs").Scan(&count).Error).To(BeNil())
			Expect(count).To(Equal(countAll("config_items", "tags->>'cluster' = 'aws'")))
		})

		It("should call config_detail", func() {
			var count int64
			Expect(tx.Raw("SELECT COUNT(*) FROM config_detail").Scan(&count).Error).To(BeNil())
			Expect(count).To(Equal(countAll("config_items", "tags->>'cluster' = 'aws'")))
		})

		It("should call config_item_summary_7d", func() {
			var count int64
			Expect(tx.Raw("SELECT COUNT(*) FROM config_item_summary_7d").Scan(&count).Error).To(BeNil())
			Expect(count).To(Equal(countAll("config_items", "")))
		})
	})

	// runCases runs each claim against the table, as each PostgREST role.
	runCases := func(table string, cases func() []grantCase) {
		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				var tx *gorm.DB

				BeforeAll(func() {
					tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				AfterAll(func() {
					Expect(tx.Commit().Error).To(BeNil())
				})

				It("lists every row when RLS is disabled", func() {
					Expect(countRows(tx, table, rls.Payload{Disable: true})).To(Equal(countAll(table, "")))
				})

				It("lists no row without grants", func() {
					Expect(countRows(tx, table, rls.Payload{})).To(BeZero())
				})

				It("lists every claim's rows", func() {
					for _, tc := range cases() {
						Expect(countRows(tx, table, tc.payload())).To(Equal(tc.expected()), tc.name)
					}
				})
			})
		}
	}

	var _ = Describe("config_items query", func() {
		runCases("config_items", func() []grantCase {
			aws := countAll("config_items", "tags->>'cluster' = 'aws'")
			return []grantCase{
				{"all rows", func() rls.Payload { return rls.Payload{Config: rls.AllRows()} }, func() int64 { return countAll("config_items", "") }},
				{"no rows", func() rls.Payload { return rls.Payload{Config: rls.NoRows()} }, func() int64 { return 0 }},
				{"a grant on another type", func() rls.Payload { return rls.Payload{Component: rls.AllRows()} }, func() int64 { return 0 }},
				{"one Scope", func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope})} }, func() int64 { return aws }},
				{"a whole-type Scope", func() rls.Payload { return rls.Payload{Config: grants([]string{allConfigsScope})} }, func() int64 { return countAll("config_items", "") }},
				{"by agent", func() rls.Payload { return rls.Payload{Config: grants([]string{localAgentScope})} }, func() int64 {
					return countAll("config_items", "agent_id = ?", uuid.Nil)
				}},
				{"every Scope of a grant", func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope, eksScope})} }, func() int64 {
					return countAll("config_items", "tags->>'cluster' = 'aws' AND name = ?", *dummy.EKSCluster.Name)
				}},
				{"a whole-type Scope narrowed", func() rls.Payload { return rls.Payload{Config: grants([]string{allConfigsScope, awsScope})} }, func() int64 { return aws }},
				{"any grant", func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope}, []string{demoScope})} }, func() int64 {
					return countAll("config_items", "tags->>'cluster' IN ('aws', 'demo')")
				}},
				{"a grant naming an unbuilt Scope fails whole", func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope, unbuiltScope})} }, func() int64 { return 0 }},
				{"other grants still apply", func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope}, []string{unbuiltScope})} }, func() int64 { return aws }},
				{"a Scope of another type", func() rls.Payload { return rls.Payload{Config: grants([]string{logisticsComponentScope})} }, func() int64 { return 0 }},
			}
		})
	})

	var _ = Describe("config child tables", Ordered, func() {
		var (
			tx        *gorm.DB
			rawCostID = uuid.New()
		)

		BeforeAll(func() {
			// Fixtures seed the compacted series; config_costs has its own policy and is
			// otherwise empty, so give it a row attached to an aws-cluster config item.
			bucket := time.Now().UTC().Truncate(time.Hour)
			Expect(DefaultContext.DB().Create(&models.ConfigCost{
				ID: rawCostID, ConfigID: dummy.KubernetesNodeA.ID, SourceKey: "rls-raw-cost",
				PeriodStart: bucket.Add(-time.Hour), PeriodEnd: bucket,
				Grain: models.ConfigCostLevel1, ChargeCategory: "Usage", BillingCurrency: "USD",
				BilledCost: decimal.NewFromInt(1), EffectiveCost: decimal.NewFromInt(1),
				Fingerprint: "rls-raw-cost",
			}).Error).To(Succeed())

			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})
			Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(BeNil())
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
			Expect(DefaultContext.DB().Delete(&models.ConfigCost{}, rawCostID).Error).To(Succeed())
		})

		awsOnly := func() rls.Payload { return rls.Payload{Config: grants([]string{awsScope})} }

		for _, table := range []string{"config_changes", "config_analysis", "config_costs", "config_cost_compact", "config_component_relationships"} {
			It("lists "+table+" of readable configs", func() {
				expected := countAll(table, "EXISTS (SELECT 1 FROM config_items c WHERE c.id = "+table+".config_id AND c.tags->>'cluster' = 'aws')")
				if table == "config_costs" || table == "config_cost_compact" {
					Expect(expected).To(BeNumerically(">", 0))
				}
				Expect(countRows(tx, table, awsOnly())).To(Equal(expected))
				Expect(countRows(tx, table, rls.Payload{Config: rls.NoRows()})).To(BeZero())
			})
		}

		It("lists config_relationships whose configs are both readable", func() {
			expected := countAll("config_relationships", `EXISTS (SELECT 1 FROM config_items c WHERE c.id = config_relationships.config_id AND c.tags->>'cluster' = 'aws')
				AND EXISTS (SELECT 1 FROM config_items c WHERE c.id = config_relationships.related_id AND c.tags->>'cluster' = 'aws')`)
			Expect(countRows(tx, "config_relationships", awsOnly())).To(Equal(expected))
		})
	})

	var _ = Describe("components query", func() {
		runCases("components", func() []grantCase {
			return []grantCase{
				{"all rows", func() rls.Payload { return rls.Payload{Component: rls.AllRows()} }, func() int64 { return countAll("components", "") }},
				{"by name", func() rls.Payload { return rls.Payload{Component: grants([]string{logisticsComponentScope})} }, func() int64 {
					return countAll("components", "name = ?", dummy.Logistics.Name)
				}},
				{"by agent", func() rls.Payload { return rls.Payload{Component: grants([]string{gcpComponentScope})} }, func() int64 {
					return countAll("components", "agent_id = ?", dummy.GCPAgent.ID)
				}},
				{"any grant", func() rls.Payload {
					return rls.Payload{Component: grants([]string{logisticsComponentScope}, []string{gcpComponentScope})}
				}, func() int64 {
					return countAll("components", "name = ? OR agent_id = ?", dummy.Logistics.Name, dummy.GCPAgent.ID)
				}},
				{"a grant on configs", func() rls.Payload { return rls.Payload{Config: rls.AllRows()} }, func() int64 { return 0 }},
			}
		})
	})

	var _ = Describe("playbooks query", func() {
		runCases("playbooks", func() []grantCase {
			return []grantCase{
				{"all rows", func() rls.Payload { return rls.Payload{Playbook: rls.AllRows()} }, func() int64 { return countAll("playbooks", "") }},
				{"by name", func() rls.Payload { return rls.Payload{Playbook: grants([]string{echoPlaybookScope})} }, func() int64 {
					return countAll("playbooks", "name = ?", dummy.EchoConfig.Name)
				}},
				{"by namespace", func() rls.Payload { return rls.Payload{Playbook: grants([]string{mcPlaybooksScope})} }, func() int64 {
					return countAll("playbooks", "namespace = ?", dummy.EchoConfig.Namespace)
				}},
			}
		})
	})

	var _ = Describe("canaries query", func() {
		runCases("canaries", func() []grantCase {
			return []grantCase{
				{"all rows", func() rls.Payload { return rls.Payload{Canary: rls.AllRows()} }, func() int64 { return countAll("canaries", "") }},
				{"by name", func() rls.Payload { return rls.Payload{Canary: grants([]string{logisticsAPICanaryScope})} }, func() int64 {
					return countAll("canaries", "name = ?", dummy.LogisticsAPICanary.Name)
				}},
				{"by agent", func() rls.Payload { return rls.Payload{Canary: grants([]string{gcpCanaryScope})} }, func() int64 {
					return countAll("canaries", "agent_id = ?", dummy.GCPAgent.ID)
				}},
			}
		})
	})

	var _ = Describe("checks query", func() {
		runCases("checks", func() []grantCase {
			return []grantCase{
				{"all rows", func() rls.Payload { return rls.Payload{Check: rls.AllRows()} }, func() int64 { return countAll("checks", "") }},
				{"through their canary", func() rls.Payload { return rls.Payload{Canary: grants([]string{logisticsAPICanaryScope})} }, func() int64 {
					return countAll("checks", "canary_id = ?", dummy.LogisticsAPICanary.ID)
				}},
				{"through every canary", func() rls.Payload { return rls.Payload{Canary: rls.AllRows()} }, func() int64 { return countAll("checks", "") }},
				{"through their own Scopes", func() rls.Payload { return rls.Payload{Check: grants([]string{apiHealthCheckScope})} }, func() int64 { return 1 }},
				{"through either", func() rls.Payload {
					return rls.Payload{Check: grants([]string{apiHealthCheckScope}), Canary: grants([]string{gcpCanaryScope})}
				}, func() int64 {
					return countAll("checks", "id = ? OR canary_id IN (SELECT id FROM canaries WHERE agent_id = ?)", dummy.LogisticsAPIHealthHTTPCheck.ID, dummy.GCPAgent.ID)
				}},
			}
		})
	})

	var _ = Describe("playbook_runs query", func() {
		runCases("playbook_runs", func() []grantCase {
			return []grantCase{
				{"every playbook, config and check", func() rls.Payload {
					return rls.Payload{Playbook: rls.AllRows(), Config: rls.AllRows(), Canary: rls.AllRows()}
				}, func() int64 { return countAll("playbook_runs", "") }},
				{"one playbook", func() rls.Payload {
					return rls.Payload{Playbook: grants([]string{echoPlaybookScope}), Config: rls.AllRows(), Canary: rls.AllRows()}
				}, func() int64 {
					return countAll("playbook_runs", "playbook_id = ?", dummy.EchoConfig.ID)
				}},
				{"playbooks without their configs", func() rls.Payload {
					return rls.Payload{Playbook: rls.AllRows(), Canary: rls.AllRows()}
				}, func() int64 {
					return countAll("playbook_runs", "config_id IS NULL")
				}},
				{"configs without their playbooks", func() rls.Payload { return rls.Payload{Config: rls.AllRows()} }, func() int64 { return 0 }},
			}
		})
	})

	var _ = Describe("INSERT QUERY", func() {
		var tx *gorm.DB

		// PostgreSQL RLS policies without an explicit WITH CHECK clause use the USING clause for writes too.
		BeforeEach(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin()
			Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(BeNil())
		})

		AfterEach(func() {
			Expect(tx.Rollback().Error).To(BeNil())
		})

		newConfig := func(name string) *models.ConfigItem {
			return &models.ConfigItem{
				ID:          uuid.New(),
				ConfigClass: "TestClass",
				Type:        lo.ToPtr("Test::Type"),
				Name:        lo.ToPtr(name),
				Tags:        types.JSONStringMap{"cluster": "aws"},
			}
		}

		It("allows INSERT to a subject granted every config", func() {
			Expect((rls.Payload{Config: rls.AllRows()}).SetPostgresSessionRLS(tx)).To(Succeed())
			Expect(tx.Create(newConfig("test-config-insert-allowed")).Error).To(Succeed())
		})

		It("denies INSERT through a Scope: the row is checked before it's matched to its Scopes", func() {
			Expect((rls.Payload{Config: grants([]string{awsScope})}).SetPostgresSessionRLS(tx)).To(Succeed())
			err := tx.Create(newConfig("test-config-insert-denied")).Error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("new row violates row-level security policy"))
		})

		It("allows INSERT through a whole-type Scope", func() {
			Expect((rls.Payload{Config: grants([]string{allConfigsScope})}).SetPostgresSessionRLS(tx)).To(Succeed())
			Expect(tx.Create(newConfig("test-config-insert-whole-type")).Error).To(Succeed())
		})
	})

	var _ = Describe("membership changes", Ordered, func() {
		var item models.ConfigItem

		BeforeAll(func() {
			item = models.ConfigItem{ID: uuid.New(), ConfigClass: "Test", Type: lo.ToPtr("Test::Type"), Name: lo.ToPtr("rls-moves"), Tags: types.JSONStringMap{"cluster": "demo"}}
			Expect(DefaultContext.DB().Create(&item).Error).To(Succeed())
		})

		AfterAll(func() {
			Expect(DefaultContext.DB().Delete(&item).Error).To(Succeed())
		})

		visible := func(scope string) bool {
			GinkgoHelper()
			tx := DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})
			defer tx.Rollback()
			Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(Succeed())
			Expect((rls.Payload{Config: grants([]string{scope})}).SetPostgresSessionRLS(tx)).To(Succeed())

			var count int64
			Expect(tx.Table("config_items").Where("id = ?", item.ID).Count(&count).Error).To(Succeed())
			return count == 1
		}

		It("lists a new resource through its Scopes as soon as it's saved", func() {
			Expect(visible(demoScope)).To(BeTrue())
			Expect(visible(awsScope)).To(BeFalse())
		})

		It("moves a changed resource between Scopes in the transaction that changes it", func() {
			tx := DefaultContext.DB().Begin()
			Expect(tx.Model(&models.ConfigItem{}).Where("id = ?", item.ID).
				Update("tags", types.JSONStringMap{"cluster": "aws"}).Error).To(Succeed())
			Expect(visible(demoScope)).To(BeTrue(), "the previous membership holds until the change commits")
			Expect(visible(awsScope)).To(BeFalse())
			Expect(tx.Commit().Error).To(Succeed())

			Expect(visible(demoScope)).To(BeFalse())
			Expect(visible(awsScope)).To(BeTrue())
		})
	})

	var _ = Describe("views query", func() {
		var (
			tx                  *gorm.DB
			totalViews          int64
			podsViewCount       int64
			devDashboardCount   int64
			podsAndDevDashboard int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.View{}).Where("deleted_at IS NULL").Count(&totalViews).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("name = ? AND deleted_at IS NULL", dummy.PodView.Name).Model(&models.View{}).Count(&podsViewCount).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("name = ? AND deleted_at IS NULL", dummy.ViewDev.Name).Model(&models.View{}).Count(&devDashboardCount).Error).To(BeNil())
			podsAndDevDashboard = podsViewCount + devDashboardCount

			Expect(totalViews).To(BeNumerically(">", 0), "No views found in test data")
			Expect(podsViewCount).To(BeNumerically(">", 0), "No pods view found")
			Expect(devDashboardCount).To(BeNumerically(">", 0), "No dev dashboard view found")
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
						Expect(tx.Model(&models.View{}).Where("deleted_at IS NULL").Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no permissions (empty scopes array)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("no permissions (non-existent view)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Names: []string{"non-existent-view"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("access specific view by name (pods)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name}},
							},
						},
						expectedCount: &podsViewCount,
					}),
					Entry("access specific view by name (Dev Dashboard)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.ViewDev.Name}},
							},
						},
						expectedCount: &devDashboardCount,
					}),
					Entry("access specific view by ID", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{ID: dummy.PodView.ID.String()},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("wildcard name (match all views)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalViews,
					}),
					Entry("wildcard ID (match all views)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{ID: "*"},
							},
						},
						expectedCount: &totalViews,
					}),
					Entry("multiple view names (OR within names array)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name, dummy.ViewDev.Name}},
							},
						},
						expectedCount: &podsAndDevDashboard,
					}),
					Entry("mixed scope criteria (OR logic between scopes)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name}},
								{Names: []string{dummy.ViewDev.Name}},
							},
						},
						expectedCount: &podsAndDevDashboard,
					}),
					Entry("ID + matching name (AND logic - should grant access)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									ID:    dummy.PodView.ID.String(),
									Names: []string{dummy.PodView.Name},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("ID + non-matching name (AND logic - should deny access)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									ID:    dummy.PodView.ID.String(),
									Names: []string{"wrong-name"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with different IDs (OR logic)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{ID: dummy.PodView.ID.String()},
								{ID: dummy.ViewDev.ID.String()},
							},
						},
						expectedCount: lo.ToPtr(int64(2)),
					}),
					Entry("empty string in names array (should deny access)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{""}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("case sensitivity - uppercase name (should deny access)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{strings.ToUpper(dummy.PodView.Name)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("duplicate scopes (should work same as single)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name}},
								{Names: []string{dummy.PodView.Name}}, // duplicate
							},
						},
						expectedCount: &podsViewCount,
					}),
					Entry("very long names list (stress test)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Names: append(
										[]string{dummy.PodView.Name},
										func() []string {
											names := make([]string, 99)
											for i := range names {
												names[i] = fmt.Sprintf("non-existent-view-%d", i)
											}
											return names
										}()...,
									),
								},
							},
						},
						expectedCount: &podsViewCount,
					}),
					Entry("whitespace-only name", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"   "}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("extremely long name string", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{strings.Repeat("a", 1000)}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard in middle", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"pod*view"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("name with wildcard prefix", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"*view"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("multiple scopes with overlapping results", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name}},
								{ID: dummy.PodView.ID.String()},
							},
						},
						expectedCount: &podsViewCount,
					}),
					Entry("empty scope object", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("wrong ID (should deny access)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{ID: "00000000-0000-0000-0000-000000000000"},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("tags only in scope (should deny access - views don't support tags)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Tags: map[string]string{"environment": "production"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // Should deny because views don't support tags
					}),
					Entry("agents only in scope (should deny access - views don't support agents)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Agents: []string{"00000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // Should deny because views don't support agents
					}),
					Entry("tags and agents only in scope (should deny access - no applicable fields)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Tags:   map[string]string{"environment": "production"},
									Agents: []string{"00000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: lo.ToPtr(int64(0)), // Should deny because views support neither tags nor agents
					}),
					Entry("valid name + irrelevant tags (tags should be ignored)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Names: []string{dummy.PodView.Name},
									Tags:  map[string]string{"environment": "production"},
								},
							},
						},
						expectedCount: &podsViewCount, // Tags should be ignored for views
					}),
					Entry("valid name + irrelevant agents (agents should be ignored)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Names:  []string{dummy.PodView.Name},
									Agents: []string{"00000000-0000-0000-0000-000000000000"},
								},
							},
						},
						expectedCount: &podsViewCount, // Agents should be ignored for views
					}),
					Entry("mixed valid and invalid names", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{
									Names: []string{
										dummy.PodView.Name,
										"non-existent-1",
										dummy.ViewDev.Name,
										"non-existent-2",
									},
								},
							},
						},
						expectedCount: &podsAndDevDashboard,
					}),
					Entry("newline in name", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"pods\nmalicious"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("special characters in name (unicode)", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"view-名前-🚀"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("very many scopes (stress test)", testCase{
						rlsPayload: rls.Payload{
							View: append(
								[]rls.Scope{{Names: []string{dummy.PodView.Name}}},
								func() []rls.Scope {
									scopes := make([]rls.Scope, 49)
									for i := range scopes {
										scopes[i] = rls.Scope{
											Names: []string{fmt.Sprintf("non-existent-%d", i)},
										}
									}
									return scopes
								}()...,
							),
						},
						expectedCount: &podsViewCount,
					}),
				)
			})
		}
	})

	var _ = Describe("view_panels query", func() {
		var (
			tx                *gorm.DB
			totalViewPanels   int64
			podViewPanelCount int64
			devViewPanelCount int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.ViewPanel{}).Count(&totalViewPanels).Error).To(BeNil())
			Expect(totalViewPanels).To(Equal(int64(2)), "Expected exactly 2 view panels in test data")

			// Count panels for PodView specifically
			Expect(DefaultContext.DB().Where("view_id = ?", dummy.PodView.ID).Model(&models.ViewPanel{}).Count(&podViewPanelCount).Error).To(BeNil())
			Expect(podViewPanelCount).To(Equal(int64(1)), "Expected exactly 1 panel for PodView")

			// Count panels for DevView specifically
			Expect(DefaultContext.DB().Where("view_id = ?", dummy.ViewDev.ID).Model(&models.ViewPanel{}).Count(&devViewPanelCount).Error).To(BeNil())
			Expect(devViewPanelCount).To(Equal(int64(1)), "Expected exactly 1 panel for DevView")
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
						Expect(tx.Model(&models.ViewPanel{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("user has permission to PodView - should see 1 view panel", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("user has no view permissions - should see 0 view panels", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("user has permission to non-existent view - should see 0 view panels", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"non-existent-view"}},
							},
						},
						expectedCount: lo.ToPtr(int64(0)),
					}),
					Entry("user has permission to ViewDev - should see 1 view panel", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.ViewDev.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("user has permission to both views - should see 2 view panels", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{dummy.PodView.Name, dummy.ViewDev.Name}},
							},
						},
						expectedCount: lo.ToPtr(int64(2)),
					}),
					Entry("user has wildcard permission - should see all panels", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{Names: []string{"*"}},
							},
						},
						expectedCount: &totalViewPanels,
					}),
					Entry("user has permission by view ID - should see panel", testCase{
						rlsPayload: rls.Payload{
							View: []rls.Scope{
								{ID: dummy.PodView.ID.String()},
							},
						},
						expectedCount: lo.ToPtr(int64(1)),
					}),
					Entry("RLS disabled - should see all panels", testCase{
						rlsPayload: rls.Payload{
							Disable: true,
						},
						expectedCount: &totalViewPanels,
					}),
				)
			})
		}
	})
})
