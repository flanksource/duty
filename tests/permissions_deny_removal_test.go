package tests

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/functions"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/views"
)

// Serial because it puts permissions.deny back, as on a database that hasn't migrated yet.
var _ = Describe("permissions deny removal", Serial, Ordered, func() {
	const script = "permissions_deny_removal.sql"

	var allowPermission, denyPermission, allowRole, denyRole uuid.UUID

	run := func(path string, scripts func() (map[string]string, error)) {
		GinkgoHelper()
		all, err := scripts()
		Expect(err).ToNot(HaveOccurred())
		Expect(all).To(HaveKey(path))
		Expect(DefaultContext.DB().Exec(all[path]).Error).To(Succeed())
	}

	hasDenyColumn := func() bool {
		GinkgoHelper()
		var exists bool
		Expect(DefaultContext.DB().Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'permissions' AND column_name = 'deny')`).Scan(&exists).Error).To(Succeed())
		return exists
	}

	exists := func(table string, id uuid.UUID) bool {
		GinkgoHelper()
		var count int64
		Expect(DefaultContext.DB().Table(table).Where("id = ?", id).Count(&count).Error).To(Succeed())
		return count == 1
	}

	BeforeAll(func() {
		Expect(DefaultContext.DB().Exec("ALTER TABLE permissions ADD COLUMN deny boolean DEFAULT false").Error).To(Succeed())

		for _, id := range []*uuid.UUID{&allowPermission, &denyPermission} {
			*id = uuid.New()
			Expect(DefaultContext.DB().Create(&models.Permission{
				ID:      *id,
				Name:    "deny-removal-" + id.String(),
				Action:  "read",
				Object:  "catalog",
				Subject: "deny-removal",
				Source:  "test",
			}).Error).To(Succeed())
		}
		Expect(DefaultContext.DB().Exec("UPDATE permissions SET deny = true WHERE id = ?", denyPermission).Error).To(Succeed())

		for id, rules := range map[*uuid.UUID]string{
			&allowRole: `[{"name": "read", "action": "read", "deny": false}]`,
			&denyRole:  `[{"name": "read", "action": "read"}, {"name": "no-delete", "action": "delete", "deny": true}]`,
		} {
			*id = uuid.New()
			Expect(DefaultContext.DB().Create(&models.Role{
				ID:        *id,
				Name:      "deny-removal-" + id.String(),
				Namespace: "default",
				Source:    "test",
				Rules:     []byte(rules),
			}).Error).To(Succeed())
		}

		Expect(DefaultContext.DB().Exec(`INSERT INTO casbin_rule (ptype, v0, v1, v2, v3, v4, v5) VALUES
			('p', 'deny-removal', 'catalog', 'read', 'allow', '', 'na'),
			('p', 'deny-removal', 'catalog', 'delete', 'deny', '', 'na')`).Error).To(Succeed())

		run(script, functions.GetFunctions)
	})

	AfterAll(func() {
		Expect(DefaultContext.DB().Exec("ALTER TABLE permissions DROP COLUMN IF EXISTS deny").Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM permissions WHERE id IN ?", []uuid.UUID{allowPermission, denyPermission}).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM roles WHERE id IN ?", []uuid.UUID{allowRole, denyRole}).Error).To(Succeed())
		Expect(DefaultContext.DB().Exec("DELETE FROM casbin_rule WHERE v0 = 'deny-removal'").Error).To(Succeed())
		run("036_permission.sql", views.GetViews)
	})

	It("drops the deny column", func() {
		Expect(hasDenyColumn()).To(BeFalse())
	})

	It("deletes the Permissions stored with deny", func() {
		Expect(exists("permissions", denyPermission)).To(BeFalse())
		Expect(exists("permissions", allowPermission)).To(BeTrue())
	})

	It("deletes the Roles with a deny rule", func() {
		Expect(exists("roles", denyRole)).To(BeFalse())
		Expect(exists("roles", allowRole)).To(BeTrue())
	})

	It("deletes the stored Casbin deny policies", func() {
		var actions []string
		Expect(DefaultContext.DB().Raw("SELECT v2 FROM casbin_rule WHERE v0 = 'deny-removal'").Scan(&actions).Error).To(Succeed())
		Expect(actions).To(ConsistOf("read"))
	})

	It("recreates the permission views without deny", func() {
		run("036_permission.sql", views.GetViews)

		var columns []string
		Expect(DefaultContext.DB().Raw(`SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name IN ('permissions_summary', 'permissions_group_summary')`).
			Scan(&columns).Error).To(Succeed())
		Expect(columns).ToNot(BeEmpty())
		Expect(columns).ToNot(ContainElement("deny"))

		var summary []models.Permission
		Expect(DefaultContext.DB().Table("permissions_summary").Where("id = ?", allowPermission).Find(&summary).Error).To(Succeed())
		Expect(summary).To(HaveLen(1))
	})

	It("does nothing when run again", func() {
		run(script, functions.GetFunctions)
		Expect(exists("permissions", allowPermission)).To(BeTrue())
		Expect(exists("roles", allowRole)).To(BeTrue())
	})
})
