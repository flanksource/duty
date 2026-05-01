package tests

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = Describe("Config Access Summary View", Ordered, func() {
	It("should surface mission control access summaries from dummy fixtures", func() {
		var accessSummaries []models.ConfigAccessSummary
		err := DefaultContext.DB().Where("config_id = ?", dummy.MissionControlNamespace.ID).
			Order("last_signed_in_at DESC").
			Find(&accessSummaries).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(accessSummaries).To(HaveLen(6))

		normalized := make([]models.ConfigAccessSummary, 0, len(accessSummaries))
		for _, summary := range accessSummaries {
			normalized = append(normalized, models.ConfigAccessSummary{
				User: summary.User,
				Role: summary.Role,
			})
		}

		expected := []models.ConfigAccessSummary{
			{
				User: dummy.JohnDoeExternalUser.Name,
				Role: dummy.MissionControlNamespaceViewerRole.Name,
			},
			{User: dummy.JohnDoeExternalUser.Name},
			{User: dummy.AliceExternalUser.Name},
			{User: dummy.BobExternalUser.Name},
			{User: dummy.CharlieExternalUser.Name},
			{User: dummy.MissionControlEmptyGroup.Name},
		}

		Expect(normalized).To(ConsistOf(expected))
	})

	It("should surface group grants even when the group has no resolved members", func() {
		var emptyGroupRow models.ConfigAccessSummary
		err := DefaultContext.DB().
			Where("config_id = ? AND external_group_id = ?",
				dummy.MissionControlNamespace.ID, dummy.MissionControlEmptyGroup.ID).
			First(&emptyGroupRow).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(emptyGroupRow.User).To(Equal(dummy.MissionControlEmptyGroup.Name))
		Expect(emptyGroupRow.UserType).To(Equal("group"))
		Expect(emptyGroupRow.ExternalUserID).To(Equal(uuid.Nil))
	})

	It("should not fan out rows for users with multiple access log entries", func() {
		// JohnDoe has one access log entry; John's direct grant should still yield a single row.
		var johnDirectRows []models.ConfigAccessSummary
		err := DefaultContext.DB().
			Where("config_id = ? AND external_user_id = ? AND external_group_id IS NULL",
				dummy.MissionControlNamespace.ID, dummy.JohnDoeExternalUser.ID).
			Find(&johnDirectRows).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(johnDirectRows).To(HaveLen(1))
		Expect(johnDirectRows[0].LastSignedInAt).ToNot(BeNil())
	})

	It("should summarize external group members and permissions", func() {
		deletedAt := dummy.DummyCreatedAt.Add(48 * time.Hour)
		deletedGroup := models.ExternalGroup{
			ID:        uuid.New(),
			ScraperID: dummy.KubeScrapeConfig.ID,
			Tenant:    "flanksource",
			Name:      "soft-deleted-member-group",
			GroupType: "group",
			CreatedAt: dummy.DummyCreatedAt,
		}
		deletedUser := models.ExternalUser{
			ID:        uuid.New(),
			ScraperID: dummy.KubeScrapeConfig.ID,
			Tenant:    "flanksource",
			Name:      "Soft Deleted Member",
			UserType:  "user",
			CreatedAt: dummy.DummyCreatedAt,
		}
		deletedMembership := models.ExternalUserGroup{
			ExternalUserID:  deletedUser.ID,
			ExternalGroupID: deletedGroup.ID,
			ScraperID:       dummy.KubeScrapeConfig.ID,
			CreatedAt:       dummy.DummyCreatedAt,
			DeletedAt:       &deletedAt,
		}

		Expect(DefaultContext.DB().Create(&deletedGroup).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Create(&deletedUser).Error).ToNot(HaveOccurred())
		Expect(DefaultContext.DB().Create(&deletedMembership).Error).ToNot(HaveOccurred())

		type externalGroupSummary struct {
			ID               uuid.UUID
			MembersCount     int64
			PermissionsCount int64
		}

		var summaries []externalGroupSummary
		err := DefaultContext.DB().
			Table("external_group_summary").
			Where("id IN ?", []uuid.UUID{
				dummy.MissionControlAdminsGroup.ID,
				dummy.MissionControlReadersGroup.ID,
				dummy.MissionControlEmptyGroup.ID,
				deletedGroup.ID,
			}).
			Find(&summaries).Error
		Expect(err).ToNot(HaveOccurred())

		byID := map[uuid.UUID]externalGroupSummary{}
		for _, summary := range summaries {
			byID[summary.ID] = summary
		}

		Expect(byID[dummy.MissionControlAdminsGroup.ID].MembersCount).To(Equal(int64(2)))
		Expect(byID[dummy.MissionControlAdminsGroup.ID].PermissionsCount).To(Equal(int64(2)))
		Expect(byID[dummy.MissionControlReadersGroup.ID].MembersCount).To(Equal(int64(2)))
		Expect(byID[dummy.MissionControlReadersGroup.ID].PermissionsCount).To(Equal(int64(2)))
		Expect(byID[dummy.MissionControlEmptyGroup.ID].MembersCount).To(Equal(int64(0)))
		Expect(byID[dummy.MissionControlEmptyGroup.ID].PermissionsCount).To(Equal(int64(1)))
		Expect(byID[deletedGroup.ID].MembersCount).To(Equal(int64(0)))
		Expect(byID[deletedGroup.ID].PermissionsCount).To(Equal(int64(0)))
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
