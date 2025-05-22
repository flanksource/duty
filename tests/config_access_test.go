package tests

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("Config Access Summary View", Ordered, func() {
	var (
		user1, user2, user3, user4, user5 models.ExternalUser
		group1                            models.ExternalGroup
		configItem                        models.ConfigItem

		referenceTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	BeforeAll(func() {
		scraperID := uuid.MustParse(*dummy.KubernetesCluster.ScraperID)

		// 1. Create 5 external users
		user1 = models.ExternalUser{ID: uuid.New(), Name: "User One", Email: lo.ToPtr("user1@example.com")}
		user2 = models.ExternalUser{ID: uuid.New(), Name: "User Two", Email: lo.ToPtr("user2@example.com")}
		user3 = models.ExternalUser{ID: uuid.New(), Name: "User Three (in group)", Email: lo.ToPtr("user3@example.com")}
		user4 = models.ExternalUser{ID: uuid.New(), Name: "User Four (in group)", Email: lo.ToPtr("user4@example.com")}
		user5 = models.ExternalUser{ID: uuid.New(), Name: "User Five (no access)", Email: lo.ToPtr("user5@example.com")}

		usersToCreate := []*models.ExternalUser{&user1, &user2, &user3, &user4, &user5}
		for _, u := range usersToCreate {
			u.ScraperID = scraperID
			err := DefaultContext.DB().Create(u).Error
			Expect(err).ToNot(HaveOccurred())
		}

		// 2. Create 1 group
		group1 = models.ExternalGroup{ID: uuid.New(), Name: "Group One"}
		group1.ScraperID = scraperID
		err := DefaultContext.DB().Create(&group1).Error
		Expect(err).ToNot(HaveOccurred())

		// Add 2 external users to that group
		userGroup1 := models.ExternalUserGroup{ExternalUserID: user3.ID, ExternalGroupID: group1.ID}
		userGroup2 := models.ExternalUserGroup{ExternalUserID: user4.ID, ExternalGroupID: group1.ID}
		userGroups := []models.ExternalUserGroup{userGroup1, userGroup2}
		for _, ug := range userGroups {
			err = DefaultContext.DB().Create(&ug).Error
			Expect(err).ToNot(HaveOccurred())
		}

		// 3. Create a config access record for a given user (not in the group)
		configItem = dummy.KubernetesCluster

		configAccessUser := models.ConfigAccess{
			ID:             uuid.NewString(),
			ConfigID:       configItem.ID,
			ExternalUserID: &user1.ID,
			ScraperID:      scraperID,
		}
		err = DefaultContext.DB().Create(&configAccessUser).Error
		Expect(err).ToNot(HaveOccurred())

		// 4. Create another config access record for a group (not a user)
		configAccessGroup := models.ConfigAccess{
			ID:              uuid.NewString(),
			ConfigID:        configItem.ID,
			ExternalGroupID: &group1.ID,
			ScraperID:       scraperID,
		}
		err = DefaultContext.DB().Create(&configAccessGroup).Error
		Expect(err).ToNot(HaveOccurred())

		// 5. Add access logs for the 3 users who should have access
		accessLogUser1 := models.ConfigAccessLog{
			ExternalUserID: user1.ID,
			ConfigID:       configItem.ID,
			ScraperID:      scraperID,
			CreatedAt:      referenceTime.Add(-time.Hour),
		}
		accessLogUser3 := models.ConfigAccessLog{
			ExternalUserID: user3.ID,
			ConfigID:       configItem.ID,
			ScraperID:      scraperID,
			CreatedAt:      referenceTime.Add(-2 * time.Hour),
		}
		accessLogUser4 := models.ConfigAccessLog{
			ExternalUserID: user4.ID,
			ConfigID:       configItem.ID,
			ScraperID:      scraperID,
			CreatedAt:      referenceTime.Add(-3 * time.Hour),
		}
		accessLogsToCreate := []*models.ConfigAccessLog{&accessLogUser1, &accessLogUser3, &accessLogUser4}
		for _, al := range accessLogsToCreate {
			err = DefaultContext.DB().Create(al).Error
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("Should return access summaries ordered by last sign in time", func() {
		var accessSummaries []models.ConfigAccessSummary
		err := DefaultContext.DB().Order("last_signed_in_at DESC").Find(&accessSummaries).Error
		Expect(err).ToNot(HaveOccurred())

		Expect(len(accessSummaries)).To(Equal(3), "Expected 3 access summary records")

		Expect(accessSummaries[0].User).To(Equal(user1.Name))
		Expect(accessSummaries[0].Email).To(Equal(*user1.Email))
		Expect(accessSummaries[0].LastSignedInAt.UTC()).To(Equal(referenceTime.UTC().Add(-time.Hour)))

		Expect(accessSummaries[1].User).To(Equal(user3.Name))
		Expect(accessSummaries[1].Email).To(Equal(*user3.Email))
		Expect(accessSummaries[1].LastSignedInAt.UTC()).To(Equal(referenceTime.UTC().Add(-2 * time.Hour)))

		Expect(accessSummaries[2].User).To(Equal(user4.Name))
		Expect(accessSummaries[2].Email).To(Equal(*user4.Email))
		Expect(accessSummaries[2].LastSignedInAt.UTC()).To(Equal(referenceTime.UTC().Add(-3 * time.Hour)))
	})
})
