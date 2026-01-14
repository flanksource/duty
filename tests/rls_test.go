package tests

import (
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	"github.com/flanksource/duty/api"
	"github.com/flanksource/duty/migrate"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/rls"
)

var _ = Describe("RLS test", Ordered, ContinueOnFailure, func() {
	BeforeAll(func() {
		if os.Getenv("DUTY_DB_DISABLE_RLS") == "true" {
			Skip("RLS tests are disabled because DUTY_DB_DISABLE_RLS is set to true")
		}
	})

	var (
		tx               *gorm.DB
		totalConfigs     int64
		awsConfigs       int64
		gcpConfigs       int64
		awsOrGcpConfigs  int64
		awsConfigChanges int64
		awsScopeID       uuid.UUID
		gcpScopeID       uuid.UUID
	)

	BeforeAll(func() {
		Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
		Expect(DefaultContext.DB().Where("tags->>'cluster' = 'aws'").Model(&models.ConfigItem{}).Count(&awsConfigs).Error).To(BeNil())
		Expect(DefaultContext.DB().Where("tags->>'cluster' = 'gcp'").Model(&models.ConfigItem{}).Count(&gcpConfigs).Error).To(BeNil())
		Expect(DefaultContext.DB().Where("tags->>'cluster' IN ('aws', 'gcp')").Model(&models.ConfigItem{}).Count(&awsOrGcpConfigs).Error).To(BeNil())
		Expect(DefaultContext.DB().Table("config_changes").
			Joins("JOIN config_items ON config_items.id = config_changes.config_id").
			Where("config_items.tags->>'cluster' = 'aws'").
			Count(&awsConfigChanges).Error).To(BeNil())

		sqldb, err := DefaultContext.DB().DB()
		Expect(err).To(BeNil())

		Expect(DefaultContext.DB().Exec("DELETE FROM migration_logs").Error).To(BeNil())

		connString := DefaultContext.Value("db_url").(string)
		err = migrate.RunMigrations(sqldb, api.Config{ConnectionString: connString, EnableRLS: true})
		Expect(err).To(BeNil())

		awsScopeID = uuid.New()
		gcpScopeID = uuid.New()
		Expect(DefaultContext.DB().Exec("UPDATE config_items SET __scope = ARRAY[?]::uuid[] WHERE tags->>'cluster' = 'aws'", awsScopeID).Error).To(BeNil())
		Expect(DefaultContext.DB().Exec("UPDATE config_items SET __scope = ARRAY[?]::uuid[] WHERE tags->>'cluster' = 'gcp'", gcpScopeID).Error).To(BeNil())

		tx = DefaultContext.DB().Begin()
		Expect(tx.Exec("SET LOCAL ROLE 'postgrest_api'").Error).To(BeNil())
	})

	AfterAll(func() {
		if tx != nil {
			Expect(tx.Commit().Error).To(BeNil())
		}
	})

	It("should filter config_items by scope", func() {
		payload := rls.Payload{Scopes: []uuid.UUID{awsScopeID}}
		Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

		var count int64
		Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
		Expect(count).To(Equal(awsConfigs))
	})

	It("should allow OR behavior across scopes", func() {
		payload := rls.Payload{Scopes: []uuid.UUID{awsScopeID, gcpScopeID}}
		Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

		var count int64
		Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
		Expect(count).To(Equal(awsOrGcpConfigs))
	})

	It("should allow wildcard config access", func() {
		payload := rls.Payload{WildcardScopes: []rls.WildcardResourceScope{rls.WildcardResourceScopeConfig}}
		Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

		var count int64
		Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
		Expect(count).To(Equal(totalConfigs))
	})

	It("should inherit RLS for config_changes", func() {
		payload := rls.Payload{Scopes: []uuid.UUID{awsScopeID}}
		Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

		var count int64
		Expect(tx.Table("config_changes").Count(&count).Error).To(BeNil())
		Expect(count).To(Equal(awsConfigChanges))
	})

	It("should deny access for unknown scope", func() {
		payload := rls.Payload{Scopes: []uuid.UUID{uuid.New()}}
		Expect(payload.SetPostgresSessionRLS(tx)).To(BeNil())

		var count int64
		Expect(tx.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
		Expect(count).To(Equal(int64(0)))
	})
})
