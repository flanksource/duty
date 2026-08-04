package tests

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = Describe("External user alias mappings", Ordered, func() {
	var primaryID, duplicateID uuid.UUID
	var accessID string
	var reviewID uuid.UUID

	BeforeAll(func() {
		// The primary deliberately sorts after the duplicate. The manual merge
		// must retain the selected primary rather than the lowest UUID.
		primaryID = uuid.MustParse("ffffffff-ffff-4fff-8fff-fffffffffff1")
		duplicateID = uuid.MustParse("00000000-0000-4000-8000-000000000001")
		accessID = fmt.Sprintf("external-user-alias-test-%s", uuid.NewString())
		reviewID = uuid.New()

		email := "Duplicate.User@Example.com"
		now := time.Now()
		users := []models.ExternalUser{
			{
				ID:        primaryID,
				Name:      "Primary Alias User",
				Aliases:   pq.StringArray{"primary-provider-id"},
				ScraperID: dummy.KubeScrapeConfig.ID,
				CreatedAt: now,
			},
			{
				ID:        duplicateID,
				Name:      "Duplicate Alias User",
				Aliases:   pq.StringArray{"github://duplicate-user"},
				Email:     &email,
				ScraperID: dummy.KubeScrapeConfig.ID,
				CreatedAt: now,
			},
		}
		Expect(DefaultContext.DB().Create(&users).Error).ToNot(HaveOccurred())

		duplicateRef := duplicateID
		access := models.ConfigAccess{
			ID:             accessID,
			ScraperID:      &dummy.KubeScrapeConfig.ID,
			ConfigID:       dummy.MissionControlNamespace.ID,
			ExternalUserID: &duplicateRef,
			CreatedAt:      now,
		}
		Expect(DefaultContext.DB().Create(&access).Error).ToNot(HaveOccurred())

		Expect(DefaultContext.DB().Exec(`
			INSERT INTO access_reviews
				(id, scraper_id, config_id, external_user_id, external_role_id, created_at, source)
			VALUES (?, ?, ?, ?, ?, ?, 'test')
		`, reviewID, dummy.KubeScrapeConfig.ID, dummy.MissionControlNamespace.ID,
			duplicateID, dummy.MissionControlNamespaceViewerRole.ID, now).Error).ToNot(HaveOccurred())

		duplicateCount := 2
		duplicateLog := models.ConfigAccessLog{
			ConfigID:       dummy.MissionControlNamespace.ID,
			ExternalUserID: duplicateID,
			ScraperID:      dummy.KubeScrapeConfig.ID,
			CreatedAt:      now,
			Count:          &duplicateCount,
		}
		Expect(DefaultContext.DB().Create(&duplicateLog).Error).ToNot(HaveOccurred())

		primaryCount := 3
		primaryLog := models.ConfigAccessLog{
			ConfigID:       dummy.MissionControlNamespace.ID,
			ExternalUserID: primaryID,
			ScraperID:      dummy.KubeScrapeConfig.ID,
			CreatedAt:      now.Add(-time.Minute),
			Count:          &primaryCount,
		}
		Expect(DefaultContext.DB().Create(&primaryLog).Error).ToNot(HaveOccurred())

		membership := models.ExternalUserGroup{
			ExternalUserID:  duplicateID,
			ExternalGroupID: dummy.MissionControlReadersGroup.ID,
			ScraperID:       dummy.KubeScrapeConfig.ID,
			CreatedAt:       now,
		}
		Expect(DefaultContext.DB().Create(&membership).Error).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		Expect(DefaultContext.DB().Exec("DELETE FROM config_access WHERE id = ?", accessID).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Exec("DELETE FROM access_reviews WHERE id = ?", reviewID).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Exec("DELETE FROM config_access_logs WHERE config_id = ? AND external_user_id = ? AND scraper_id = ?", dummy.MissionControlNamespace.ID, primaryID, dummy.KubeScrapeConfig.ID).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Exec("DELETE FROM external_user_groups WHERE external_user_id = ? AND external_group_id = ? AND scraper_id = ?", primaryID, dummy.MissionControlReadersGroup.ID, dummy.KubeScrapeConfig.ID).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Exec("DELETE FROM external_users WHERE id IN ?", []uuid.UUID{primaryID, duplicateID}).Error).ToNot(HaveOccurred())
	})

	It("records discovered aliases without duplicating live canonical IDs", func() {
		var mappings []models.ExternalUserAlias
		Expect(DefaultContext.DB().
			Where("external_user_id IN ?", []uuid.UUID{primaryID, duplicateID}).
			Find(&mappings).Error).ToNot(HaveOccurred())

		byAlias := make(map[string]uuid.UUID, len(mappings))
		for _, mapping := range mappings {
			byAlias[mapping.Alias] = mapping.ExternalUserID
		}
		Expect(byAlias).To(HaveKeyWithValue("primary-provider-id", primaryID))
		Expect(byAlias).To(HaveKeyWithValue("github://duplicate-user", duplicateID))
		Expect(byAlias).NotTo(HaveKey(primaryID.String()))
		Expect(byAlias).NotTo(HaveKey(duplicateID.String()))
	})

	It("adds and normalizes a manual alias", func() {
		var mapping models.ExternalUserAlias
		Expect(DefaultContext.DB().
			Raw("SELECT * FROM add_external_user_alias(?, ?)", primaryID, "  GitHub://Primary-User  ").
			Scan(&mapping).Error).ToNot(HaveOccurred())
		Expect(mapping.ExternalUserID).To(Equal(primaryID))
		Expect(mapping.Alias).To(Equal("github://primary-user"))
		Expect(mapping.Source).To(Equal("manual"))

		var primary models.ExternalUser
		Expect(DefaultContext.DB().First(&primary, "id = ?", primaryID).Error).ToNot(HaveOccurred())
		Expect([]string(primary.Aliases)).To(ContainElement("github://primary-user"))
	})

	It("deactivates derived mappings when an alias is removed from the source array", func() {
		const alias = "provider://temporary-user"
		Expect(DefaultContext.DB().Exec(`
			UPDATE external_users
			SET aliases = array_append(COALESCE(aliases, '{}'::text[]), ?)
			WHERE id = ?
		`, alias, primaryID).Error).ToNot(HaveOccurred())

		var activeCount int64
		Expect(DefaultContext.DB().Model(&models.ExternalUserAlias{}).
			Where("external_user_id = ? AND alias = ? AND deleted_at IS NULL", primaryID, alias).
			Count(&activeCount).Error).ToNot(HaveOccurred())
		Expect(activeCount).To(Equal(int64(1)))

		Expect(DefaultContext.DB().Exec(`
			UPDATE external_users
			SET aliases = array_remove(aliases, ?)
			WHERE id = ?
		`, alias, primaryID).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Model(&models.ExternalUserAlias{}).
			Where("external_user_id = ? AND alias = ? AND deleted_at IS NULL", primaryID, alias).
			Count(&activeCount).Error).ToNot(HaveOccurred())
		Expect(activeCount).To(BeZero())
	})

	It("removes a manual alias from both the source array and lookup index", func() {
		var removed bool
		Expect(DefaultContext.DB().Raw(
			"SELECT remove_external_user_alias(?, ?) AS removed",
			primaryID, " GitHub://Primary-User ",
		).Scan(&removed).Error).ToNot(HaveOccurred())
		Expect(removed).To(BeTrue())

		var primary models.ExternalUser
		Expect(DefaultContext.DB().First(&primary, "id = ?", primaryID).Error).ToNot(HaveOccurred())
		Expect([]string(primary.Aliases)).NotTo(ContainElement("github://primary-user"))

		var activeCount int64
		Expect(DefaultContext.DB().Model(&models.ExternalUserAlias{}).
			Where("external_user_id = ? AND alias = ? AND deleted_at IS NULL", primaryID, "github://primary-user").
			Count(&activeCount).Error).ToNot(HaveOccurred())
		Expect(activeCount).To(BeZero())

		Expect(DefaultContext.DB().Raw(
			"SELECT remove_external_user_alias(?, ?) AS removed",
			primaryID, "github://primary-user",
		).Scan(&removed).Error).ToNot(HaveOccurred())
		Expect(removed).To(BeFalse())
	})

	It("keeps add and concurrent delete operations consistent", func() {
		raceID := uuid.New()
		raceUser := models.ExternalUser{
			ID:        raceID,
			Name:      "Concurrent Alias User",
			Aliases:   pq.StringArray{"provider://concurrent-user"},
			ScraperID: dummy.KubeScrapeConfig.ID,
			CreatedAt: time.Now(),
		}
		Expect(DefaultContext.DB().Create(&raceUser).Error).ToNot(HaveOccurred())

		addTx := DefaultContext.DB().Begin()
		Expect(addTx.Error).ToNot(HaveOccurred())
		Expect(addTx.Exec(
			"SELECT add_external_user_alias(?, ?)", raceID, "manual://concurrent-user",
		).Error).ToNot(HaveOccurred())

		deleteDone := make(chan error, 1)
		go func() {
			deleteDone <- DefaultContext.DB().Exec(
				"UPDATE external_users SET deleted_at = now() WHERE id = ?", raceID,
			).Error
		}()

		Consistently(deleteDone, 150*time.Millisecond).ShouldNot(Receive())
		Expect(addTx.Commit().Error).ToNot(HaveOccurred())

		var deleteErr error
		Eventually(deleteDone, 3*time.Second).Should(Receive(&deleteErr))
		Expect(deleteErr).ToNot(HaveOccurred())

		var activeCount int64
		Expect(DefaultContext.DB().Model(&models.ExternalUserAlias{}).
			Where("external_user_id = ? AND deleted_at IS NULL", raceID).
			Count(&activeCount).Error).ToNot(HaveOccurred())
		Expect(activeCount).To(BeZero())
		Expect(DefaultContext.DB().Exec("DELETE FROM external_users WHERE id = ?", raceID).Error).ToNot(HaveOccurred())
	})

	It("rejects storing a live canonical ID as its own alias", func() {
		err := DefaultContext.DB().
			Exec("SELECT add_external_user_alias(?, ?)", primaryID, primaryID.String()).Error
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("is not an alias"))
	})

	It("rejects assigning an existing alias to another active user", func() {
		err := DefaultContext.DB().
			Exec("SELECT add_external_user_alias(?, ?)", primaryID, "github://duplicate-user").Error
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already assigned"))
	})

	It("rejects a conflicting alias added directly to the source array", func() {
		err := DefaultContext.DB().Exec(`
			UPDATE external_users
			SET aliases = array_append(COALESCE(aliases, '{}'::text[]), 'github://duplicate-user')
			WHERE id = ?
		`, primaryID).Error
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already assigned"))
	})

	It("rejects canonical IDs added directly to the source aliases array", func() {
		for _, canonicalID := range []uuid.UUID{primaryID, duplicateID} {
			err := DefaultContext.DB().Exec(`
				UPDATE external_users
				SET aliases = array_append(COALESCE(aliases, '{}'::text[]), ?)
				WHERE id = ?
			`, canonicalID.String(), primaryID).Error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("canonical ID of active external user"))
		}
	})

	It("rejects assigning another active user's canonical ID as an alias", func() {
		err := DefaultContext.DB().
			Exec("SELECT add_external_user_alias(?, ?)", primaryID, duplicateID.String()).Error
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already identifies active external user"))
	})

	It("rejects creating a canonical user whose ID is already a source alias", func() {
		futureID := uuid.New()
		Expect(DefaultContext.DB().Exec(
			"SELECT add_external_user_alias(?, ?)", primaryID, futureID.String(),
		).Error).ToNot(HaveOccurred())

		user := models.ExternalUser{
			ID:        futureID,
			Name:      "Conflicting Canonical User",
			Aliases:   pq.StringArray{"provider://future-user"},
			ScraperID: dummy.KubeScrapeConfig.ID,
			CreatedAt: time.Now(),
		}
		err := DefaultContext.DB().Create(&user).Error
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("already an alias of active external user"))

		Expect(DefaultContext.DB().Exec(
			"SELECT remove_external_user_alias(?, ?)", primaryID, futureID.String(),
		).Error).ToNot(HaveOccurred())
	})

	It("merges into the explicitly selected primary and redirects old IDs", func() {
		var result struct {
			Winner uuid.UUID `gorm:"column:winner"`
		}
		Expect(DefaultContext.DB().
			Raw("SELECT merge_external_users(?, ?) AS winner", primaryID, duplicateID).
			Scan(&result).Error).ToNot(HaveOccurred())
		Expect(result.Winner).To(Equal(primaryID))

		var duplicate models.ExternalUser
		Expect(DefaultContext.DB().First(&duplicate, "id = ?", duplicateID).Error).ToNot(HaveOccurred())
		Expect(duplicate.DeletedAt).ToNot(BeNil())

		var primary models.ExternalUser
		Expect(DefaultContext.DB().First(&primary, "id = ?", primaryID).Error).ToNot(HaveOccurred())
		Expect(primary.DeletedAt).To(BeNil())
		Expect([]string(primary.Aliases)).To(ContainElements(
			duplicateID.String(),
			"github://duplicate-user",
			"duplicate.user@example.com",
		))

		var mappings []models.ExternalUserAlias
		Expect(DefaultContext.DB().
			Where("alias IN ? AND deleted_at IS NULL", []string{
				duplicateID.String(),
				"github://duplicate-user",
				"duplicate.user@example.com",
			}).
			Find(&mappings).Error).ToNot(HaveOccurred())
		Expect(mappings).To(HaveLen(3))
		for _, mapping := range mappings {
			Expect(mapping.ExternalUserID).To(Equal(primaryID))
		}

		var access models.ConfigAccess
		Expect(DefaultContext.DB().First(&access, "id = ?", accessID).Error).ToNot(HaveOccurred())
		Expect(access.ExternalUserID).ToNot(BeNil())
		Expect(*access.ExternalUserID).To(Equal(primaryID))

		var review struct {
			ExternalUserID uuid.UUID
		}
		Expect(DefaultContext.DB().Table("access_reviews").
			Select("external_user_id").
			Where("id = ?", reviewID).
			Take(&review).Error).ToNot(HaveOccurred())
		Expect(review.ExternalUserID).To(Equal(primaryID))

		var accessLog models.ConfigAccessLog
		Expect(DefaultContext.DB().First(&accessLog,
			"config_id = ? AND external_user_id = ? AND scraper_id = ?",
			dummy.MissionControlNamespace.ID, primaryID, dummy.KubeScrapeConfig.ID).Error).ToNot(HaveOccurred())
		Expect(accessLog.Count).ToNot(BeNil())
		Expect(*accessLog.Count).To(Equal(5))

		var membership models.ExternalUserGroup
		Expect(DefaultContext.DB().First(&membership,
			"external_user_id = ? AND external_group_id = ? AND scraper_id = ?",
			primaryID, dummy.MissionControlReadersGroup.ID, dummy.KubeScrapeConfig.ID).Error).ToNot(HaveOccurred())
		Expect(membership.DeletedAt).To(BeNil())
	})

	It("keeps every active lookup row represented in the source aliases array", func() {
		var inconsistent int64
		Expect(DefaultContext.DB().Raw(`
			SELECT count(*)
			FROM external_user_aliases eua
			JOIN external_users eu ON eu.id = eua.external_user_id
			WHERE eua.external_user_id IN ?
			  AND eua.deleted_at IS NULL
			  AND NOT EXISTS (
			    SELECT 1
			    FROM unnest(COALESCE(eu.aliases, '{}'::text[])) AS a
			    WHERE lower(btrim(a)) = eua.alias
			  )
		`, []uuid.UUID{primaryID, duplicateID}).Scan(&inconsistent).Error).ToNot(HaveOccurred())
		Expect(inconsistent).To(BeZero())
	})

	It("is idempotent when the merge is retried", func() {
		var result struct {
			Winner uuid.UUID `gorm:"column:winner"`
		}
		Expect(DefaultContext.DB().
			Raw("SELECT merge_external_users(?, ?) AS winner", primaryID, duplicateID).
			Scan(&result).Error).ToNot(HaveOccurred())
		Expect(result.Winner).To(Equal(primaryID))
	})
})
