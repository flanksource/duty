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

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rls"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

type scopeCase struct {
	payload       func() rls.Payload
	expectedCount *int64
}

func setRLS(tx *gorm.DB, payload rls.Payload) {
	Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())
}

func payloadNoScopes() func() rls.Payload {
	return func() rls.Payload {
		return rls.Payload{}
	}
}

func payloadWithScopes(scopeIDs ...*uuid.UUID) func() rls.Payload {
	return func() rls.Payload {
		ids := make([]uuid.UUID, 0, len(scopeIDs))
		for _, id := range scopeIDs {
			if id != nil {
				ids = append(ids, *id)
			}
		}
		return rls.Payload{Scopes: ids}
	}
}

func payloadWildcard(scope rls.WildcardResourceScope) func() rls.Payload {
	return func() rls.Payload {
		return rls.Payload{WildcardScopes: []rls.WildcardResourceScope{scope}}
	}
}

func payloadDisabled() func() rls.Payload {
	return func() rls.Payload {
		return rls.Payload{Disable: true}
	}
}

func resetScopes(db *gorm.DB, tables ...string) {
	for _, table := range tables {
		Expect(db.Exec(fmt.Sprintf("UPDATE %s SET __scope = NULL", table)).Error).To(BeNil())
	}
}

func assignScope(db *gorm.DB, table string, scopeID uuid.UUID, where string, args ...any) {
	query := fmt.Sprintf("UPDATE %s SET __scope = array_append(COALESCE(__scope, '{}'::uuid[]), ?) WHERE %s", table, where)
	Expect(db.Exec(query, append([]any{scopeID}, args...)...).Error).To(BeNil())
}

var _ = Describe("RLS scopes", Ordered, ContinueOnFailure, func() {
	BeforeAll(func() {
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			Skip("RLS tests are disabled because DUTY_DB_DISABLE_RLS is set to true")
		}
	})

	var _ = Describe("config_items", func() {
		var (
			tx           *gorm.DB
			totalConfigs int64
			awsConfigs   int64
			demoConfigs  int64
			awsOrDemo    int64
			awsScopeID   = uuid.New()
			demoScopeID  = uuid.New()
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws'").Model(&models.ConfigItem{}).Count(&awsConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' = 'demo'").Model(&models.ConfigItem{}).Count(&demoConfigs).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("tags->>'cluster' IN ('aws', 'demo')").Model(&models.ConfigItem{}).Count(&awsOrDemo).Error).To(BeNil())

			resetScopes(DefaultContext.DB(), "config_items")
			assignScope(DefaultContext.DB(), "config_items", awsScopeID, "tags->>'cluster' = ?", "aws")
			assignScope(DefaultContext.DB(), "config_items", demoScopeID, "tags->>'cluster' = ?", "demo")
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				DescribeTable("RLS scope tests",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("aws scope", scopeCase{payload: payloadWithScopes(&awsScopeID), expectedCount: &awsConfigs}),
					Entry("combined scopes", scopeCase{payload: payloadWithScopes(&awsScopeID, &demoScopeID), expectedCount: &awsOrDemo}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeConfig), expectedCount: &totalConfigs}),
					Entry("rls disabled", scopeCase{payload: payloadDisabled(), expectedCount: &totalConfigs}),
				)
			})
		}
	})

	var _ = Describe("components", func() {
		var (
			tx                  *gorm.DB
			totalComponents     int64
			agentComponents     int64
			logisticsComponents int64
			agentOrLogistics    int64
			agentScopeID        uuid.UUID
			logisticsScopeID    uuid.UUID
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Component{}).Count(&totalComponents).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ?", uuid.Nil).Model(&models.Component{}).Count(&agentComponents).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("name = ?", dummy.Logistics.Name).Model(&models.Component{}).Count(&logisticsComponents).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("agent_id = ? OR name = ?", uuid.Nil, dummy.Logistics.Name).Model(&models.Component{}).Count(&agentOrLogistics).Error).To(BeNil())

			resetScopes(DefaultContext.DB(), "components")
			agentScopeID = uuid.New()
			logisticsScopeID = uuid.New()
			assignScope(DefaultContext.DB(), "components", agentScopeID, "agent_id = ?", uuid.Nil)
			assignScope(DefaultContext.DB(), "components", logisticsScopeID, "name = ?", dummy.Logistics.Name)
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				DescribeTable("RLS scope tests",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.Component{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("agent scope", scopeCase{payload: payloadWithScopes(&agentScopeID), expectedCount: &agentComponents}),
					Entry("name scope", scopeCase{payload: payloadWithScopes(&logisticsScopeID), expectedCount: &logisticsComponents}),
					Entry("combined scopes", scopeCase{payload: payloadWithScopes(&agentScopeID, &logisticsScopeID), expectedCount: &agentOrLogistics}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeComponent), expectedCount: &totalComponents}),
				)
			})
		}
	})

	var _ = Describe("playbooks", func() {
		var (
			tx               *gorm.DB
			totalPlaybooks   int64
			combinedPlaybook int64
			echoScopeID      uuid.UUID
			restartScopeID   uuid.UUID
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Playbook{}).Count(&totalPlaybooks).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("id IN ?", []uuid.UUID{dummy.EchoConfig.ID, dummy.RestartPod.ID}).Model(&models.Playbook{}).Count(&combinedPlaybook).Error).To(BeNil())

			resetScopes(DefaultContext.DB(), "playbooks")
			echoScopeID = uuid.New()
			restartScopeID = uuid.New()
			assignScope(DefaultContext.DB(), "playbooks", echoScopeID, "id = ?", dummy.EchoConfig.ID)
			assignScope(DefaultContext.DB(), "playbooks", restartScopeID, "id = ?", dummy.RestartPod.ID)
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				DescribeTable("RLS scope tests",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.Playbook{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("echo scope", scopeCase{payload: payloadWithScopes(&echoScopeID), expectedCount: lo.ToPtr(int64(1))}),
					Entry("combined scopes", scopeCase{payload: payloadWithScopes(&echoScopeID, &restartScopeID), expectedCount: &combinedPlaybook}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopePlaybook), expectedCount: &totalPlaybooks}),
				)
			})
		}
	})

	var _ = Describe("canaries and checks", func() {
		var (
			tx               *gorm.DB
			totalCanaries    int64
			logisticsScopeID uuid.UUID
			totalChecks      int64
			logisticsChecks  int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.Canary{}).Count(&totalCanaries).Error).To(BeNil())
			Expect(DefaultContext.DB().Model(&models.Check{}).Count(&totalChecks).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("canary_id = ?", dummy.LogisticsAPICanary.ID).Model(&models.Check{}).Count(&logisticsChecks).Error).To(BeNil())

			resetScopes(DefaultContext.DB(), "canaries")
			logisticsScopeID = uuid.New()
			assignScope(DefaultContext.DB(), "canaries", logisticsScopeID, "id = ?", dummy.LogisticsAPICanary.ID)
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				DescribeTable("canary scopes",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.Canary{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("logistics scope", scopeCase{payload: payloadWithScopes(&logisticsScopeID), expectedCount: lo.ToPtr(int64(1))}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeCanary), expectedCount: &totalCanaries}),
				)

				DescribeTable("checks inherit canary RLS",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.Check{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("logistics scope", scopeCase{payload: payloadWithScopes(&logisticsScopeID), expectedCount: &logisticsChecks}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeCanary), expectedCount: &totalChecks}),
				)
			})
		}
	})

	var _ = Describe("views and panels", func() {
		var (
			tx             *gorm.DB
			totalViews     int64
			totalPanels    int64
			podScopeID     uuid.UUID
			devScopeID     uuid.UUID
			podViewPanels  int64
			devViewPanels  int64
			combinedViews  int64
			combinedPanels int64
		)

		BeforeAll(func() {
			tx = DefaultContext.DB().Session(&gorm.Session{NewDB: true}).Begin(&sql.TxOptions{ReadOnly: true})

			Expect(DefaultContext.DB().Model(&models.View{}).Where("deleted_at IS NULL").Count(&totalViews).Error).To(BeNil())
			Expect(DefaultContext.DB().Model(&models.ViewPanel{}).Count(&totalPanels).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("view_id = ?", dummy.PodView.ID).Model(&models.ViewPanel{}).Count(&podViewPanels).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("view_id = ?", dummy.ViewDev.ID).Model(&models.ViewPanel{}).Count(&devViewPanels).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("id IN ?", []uuid.UUID{dummy.PodView.ID, dummy.ViewDev.ID}).Model(&models.View{}).Count(&combinedViews).Error).To(BeNil())
			Expect(DefaultContext.DB().Where("view_id IN ?", []uuid.UUID{dummy.PodView.ID, dummy.ViewDev.ID}).Model(&models.ViewPanel{}).Count(&combinedPanels).Error).To(BeNil())

			resetScopes(DefaultContext.DB(), "views")
			podScopeID = uuid.New()
			devScopeID = uuid.New()
			assignScope(DefaultContext.DB(), "views", podScopeID, "id = ?", dummy.PodView.ID)
			assignScope(DefaultContext.DB(), "views", devScopeID, "id = ?", dummy.ViewDev.ID)
		})

		AfterAll(func() {
			Expect(tx.Commit().Error).To(BeNil())
		})

		for _, role := range []string{"postgrest_anon", "postgrest_api"} {
			Context(role, Ordered, func() {
				BeforeAll(func() {
					Expect(tx.Exec(fmt.Sprintf("SET LOCAL ROLE '%s'", role)).Error).To(BeNil())
				})

				DescribeTable("views",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.View{}).Where("deleted_at IS NULL").Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("pod view", scopeCase{payload: payloadWithScopes(&podScopeID), expectedCount: lo.ToPtr(int64(1))}),
					Entry("combined", scopeCase{payload: payloadWithScopes(&podScopeID, &devScopeID), expectedCount: &combinedViews}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeView), expectedCount: &totalViews}),
				)

				DescribeTable("view panels",
					func(tc scopeCase) {
						setRLS(tx, tc.payload())

						var count int64
						Expect(tx.Model(&models.ViewPanel{}).Count(&count).Error).To(BeNil())
						Expect(count).To(Equal(*tc.expectedCount))
					},
					Entry("no scopes", scopeCase{payload: payloadNoScopes(), expectedCount: lo.ToPtr(int64(0))}),
					Entry("pod view", scopeCase{payload: payloadWithScopes(&podScopeID), expectedCount: &podViewPanels}),
					Entry("dev view", scopeCase{payload: payloadWithScopes(&devScopeID), expectedCount: &devViewPanels}),
					Entry("combined", scopeCase{payload: payloadWithScopes(&podScopeID, &devScopeID), expectedCount: &combinedPanels}),
					Entry("wildcard", scopeCase{payload: payloadWildcard(rls.WildcardResourceScopeView), expectedCount: &totalPanels}),
				)
			})
		}
	})
})
