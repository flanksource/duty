package tests

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
			ScraperID:      &scraperID,
		}
		err = DefaultContext.DB().Create(&configAccessUser).Error
		Expect(err).ToNot(HaveOccurred())

		// 4. Create another config access record for a group (not a user)
		configAccessGroup := models.ConfigAccess{
			ID:              uuid.NewString(),
			ConfigID:        configItem.ID,
			ExternalGroupID: &group1.ID,
			ScraperID:       &scraperID,
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

var _ = Describe("External Users Aliases", Ordered, func() {
	var scraperID uuid.UUID

	BeforeAll(func() {
		scraperID = uuid.MustParse(*dummy.KubernetesCluster.ScraperID)
	})

	It("should lowercase, sort and unique aliases on insert", func() {
		user := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Lowercase User",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"Lowercase-Bob", "LOWERCASE-ALICE", "LOWERCASE-alice", "LOWERCASE-alice", "LOWERCASE-CHARLIE"},
		}
		err := DefaultContext.DB().Create(&user).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalUser
		err = DefaultContext.DB().Where("id = ?", user.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"lowercase-alice", "lowercase-bob", "lowercase-charlie"}))
	})

	It("should normalize aliases on update", func() {
		user := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Update User",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"update-initial"},
		}
		err := DefaultContext.DB().Create(&user).Error
		Expect(err).ToNot(HaveOccurred())

		err = DefaultContext.DB().Model(&user).Update("aliases", pq.StringArray{"UPDATE-ZEBRA", "Update-Apple", "update-zebra"}).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalUser
		err = DefaultContext.DB().Where("id = ?", user.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"update-apple", "update-zebra"}))
	})

	It("should handle null aliases", func() {
		user := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Null Aliases User",
			ScraperID: scraperID,
			Aliases:   nil,
		}
		err := DefaultContext.DB().Create(&user).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalUser
		err = DefaultContext.DB().Where("id = ?", user.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Aliases).To(BeNil())
	})

	It("should handle empty aliases array", func() {
		user := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Empty Aliases User",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{},
		}
		err := DefaultContext.DB().Create(&user).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalUser
		err = DefaultContext.DB().Where("id = ?", user.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{}))
	})

	It("should enforce unique aliases constraint", func() {
		aliases := pq.StringArray{"unique-alias-1", "unique-alias-2"}

		user1 := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Unique User 1",
			ScraperID: scraperID,
			Aliases:   aliases,
		}
		err := DefaultContext.DB().Create(&user1).Error
		Expect(err).ToNot(HaveOccurred())

		user2 := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Unique User 2",
			ScraperID: scraperID,
			Aliases:   aliases,
		}
		err = DefaultContext.DB().Create(&user2).Error
		Expect(err).To(HaveOccurred())
	})

	It("should enforce unique constraint case-insensitively", func() {
		user1 := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Case Unique User 1",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"CaseTest1", "CaseTest2"},
		}
		err := DefaultContext.DB().Create(&user1).Error
		Expect(err).ToNot(HaveOccurred())

		user2 := models.ExternalUser{
			ID:        uuid.New(),
			Name:      "Test Case Unique User 2",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"casetest1", "CASETEST2"},
		}
		err = DefaultContext.DB().Create(&user2).Error
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("External Roles Aliases", Ordered, func() {
	var scraperID uuid.UUID

	BeforeAll(func() {
		scraperID = uuid.MustParse(*dummy.KubernetesCluster.ScraperID)
	})

	It("should lowercase, sort and unique aliases on insert", func() {
		role := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Lowercase",
			ScraperID: &scraperID,
			Aliases:   pq.StringArray{"Role-Bob", "ROLE-ALICE", "ROLE-alice", "ROLE-alice", "ROLE-CHARLIE"},
		}
		err := DefaultContext.DB().Create(&role).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalRole
		err = DefaultContext.DB().Where("id = ?", role.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"role-alice", "role-bob", "role-charlie"}))
	})

	It("should normalize aliases on update", func() {
		role := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Update",
			ScraperID: &scraperID,
			Aliases:   pq.StringArray{"role-update-initial"},
		}
		err := DefaultContext.DB().Create(&role).Error
		Expect(err).ToNot(HaveOccurred())

		err = DefaultContext.DB().Model(&role).Update("aliases", pq.StringArray{"ROLE-UPDATE-ZEBRA", "Role-Update-Apple", "role-update-zebra"}).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalRole
		err = DefaultContext.DB().Where("id = ?", role.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"role-update-apple", "role-update-zebra"}))
	})

	It("should handle null aliases", func() {
		role := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Null Aliases",
			ScraperID: &scraperID,
			Aliases:   nil,
		}
		err := DefaultContext.DB().Create(&role).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalRole
		err = DefaultContext.DB().Where("id = ?", role.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Aliases).To(BeNil())
	})

	It("should handle empty aliases array", func() {
		role := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Empty Aliases",
			ScraperID: &scraperID,
			Aliases:   pq.StringArray{},
		}
		err := DefaultContext.DB().Create(&role).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalRole
		err = DefaultContext.DB().Where("id = ?", role.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{}))
	})

	It("should enforce unique aliases constraint", func() {
		aliases := pq.StringArray{"role-unique-alias-1", "role-unique-alias-2"}

		role1 := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Unique 1",
			ScraperID: &scraperID,
			Aliases:   aliases,
		}
		err := DefaultContext.DB().Create(&role1).Error
		Expect(err).ToNot(HaveOccurred())

		role2 := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Unique 2",
			ScraperID: &scraperID,
			Aliases:   aliases,
		}
		err = DefaultContext.DB().Create(&role2).Error
		Expect(err).To(HaveOccurred())
	})

	It("should enforce unique constraint case-insensitively", func() {
		role1 := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Case Unique 1",
			ScraperID: &scraperID,
			Aliases:   pq.StringArray{"RoleCaseTest1", "RoleCaseTest2"},
		}
		err := DefaultContext.DB().Create(&role1).Error
		Expect(err).ToNot(HaveOccurred())

		role2 := models.ExternalRole{
			ID:        uuid.New(),
			Name:      "Test Role Case Unique 2",
			ScraperID: &scraperID,
			Aliases:   pq.StringArray{"rolecasetest1", "ROLECASETEST2"},
		}
		err = DefaultContext.DB().Create(&role2).Error
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("External Groups Aliases", Ordered, func() {
	var scraperID uuid.UUID

	BeforeAll(func() {
		scraperID = uuid.MustParse(*dummy.KubernetesCluster.ScraperID)
	})

	It("should lowercase, sort and unique aliases on insert", func() {
		group := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Lowercase",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"Group-Bob", "GROUP-ALICE", "GROUP-alice", "GROUP-alice", "GROUP-CHARLIE"},
		}
		err := DefaultContext.DB().Create(&group).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalGroup
		err = DefaultContext.DB().Where("id = ?", group.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"group-alice", "group-bob", "group-charlie"}))
	})

	It("should normalize aliases on update", func() {
		group := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Update",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"group-update-initial"},
		}
		err := DefaultContext.DB().Create(&group).Error
		Expect(err).ToNot(HaveOccurred())

		err = DefaultContext.DB().Model(&group).Update("aliases", pq.StringArray{"GROUP-UPDATE-ZEBRA", "Group-Update-Apple", "group-update-zebra"}).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalGroup
		err = DefaultContext.DB().Where("id = ?", group.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{"group-update-apple", "group-update-zebra"}))
	})

	It("should handle null aliases", func() {
		group := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Null Aliases",
			ScraperID: scraperID,
			Aliases:   nil,
		}
		err := DefaultContext.DB().Create(&group).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalGroup
		err = DefaultContext.DB().Where("id = ?", group.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(fetched.Aliases).To(BeNil())
	})

	It("should handle empty aliases array", func() {
		group := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Empty Aliases",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{},
		}
		err := DefaultContext.DB().Create(&group).Error
		Expect(err).ToNot(HaveOccurred())

		var fetched models.ExternalGroup
		err = DefaultContext.DB().Where("id = ?", group.ID).First(&fetched).Error
		Expect(err).ToNot(HaveOccurred())
		Expect([]string(fetched.Aliases)).To(Equal([]string{}))
	})

	It("should enforce unique aliases constraint", func() {
		aliases := pq.StringArray{"group-unique-alias-1", "group-unique-alias-2"}

		group1 := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Unique 1",
			ScraperID: scraperID,
			Aliases:   aliases,
		}
		err := DefaultContext.DB().Create(&group1).Error
		Expect(err).ToNot(HaveOccurred())

		group2 := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Unique 2",
			ScraperID: scraperID,
			Aliases:   aliases,
		}
		err = DefaultContext.DB().Create(&group2).Error
		Expect(err).To(HaveOccurred())
	})

	It("should enforce unique constraint case-insensitively", func() {
		group1 := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Case Unique 1",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"GroupCaseTest1", "GroupCaseTest2"},
		}
		err := DefaultContext.DB().Create(&group1).Error
		Expect(err).ToNot(HaveOccurred())

		group2 := models.ExternalGroup{
			ID:        uuid.New(),
			Name:      "Test Group Case Unique 2",
			ScraperID: scraperID,
			Aliases:   pq.StringArray{"groupcasetest1", "GROUPCASETEST2"},
		}
		err = DefaultContext.DB().Create(&group2).Error
		Expect(err).To(HaveOccurred())
	})
})
