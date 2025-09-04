package tests

import (
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"

	dutyRBAC "github.com/flanksource/duty/rbac"
	"github.com/flanksource/duty/rbac/policy"
)

type Info struct {
	Tables    pq.StringArray `gorm:"type:[]text"`
	Views     pq.StringArray `gorm:"type:[]text"`
	Functions pq.StringArray `gorm:"type:[]text"`
}

func (info *Info) Get(db *gorm.DB) error {
	sql := `
	SELECT tables, views, functions
	FROM (SELECT array_agg(information_schema.views.table_name) AS views
					FROM   information_schema.views
			WHERE  information_schema.views.table_schema = any (current_schemas(false)) AND table_name not like 'pg_%'
		)
		t,
		(SELECT array_agg(information_schema.tables.table_name) AS tables
			FROM   information_schema."tables"
			WHERE  information_schema.tables.table_schema = any (
						current_schemas(false) )
						AND information_schema.tables.table_type = 'BASE TABLE'
						AND table_name NOT LIKE 'view_%') v,
		(SELECT array_agg(proname) AS functions
			FROM   pg_proc p
						INNER JOIN pg_namespace ns
										ON ( p.pronamespace = ns.oid )
			WHERE  ns.nspname = 'public'
						AND probin IS NULL
						AND probin IS NULL
						AND proretset IS TRUE) f
		`
	return db.Raw(sql).Scan(info).Error
}

var _ = Describe("Authorization", func() {
	It("Should cover all db objects", func() {
		info := &Info{}
		if err := info.Get(DefaultContext.DB()); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(len(info.Functions)).To(BeNumerically(">", 0))

		for _, table := range append(info.Views, info.Tables...) {
			Expect(dutyRBAC.GetObjectByTable(table)).NotTo(BeEmpty(), table)
		}
		for _, function := range info.Functions {
			Expect(dutyRBAC.GetObjectByTable("rpc/"+function)).NotTo(BeEmpty(), function)
		}
	})

	It("Should correctly return perms for a user", func() {
		err := dutyRBAC.Init(DefaultContext, []string{uuid.Nil.String()})
		Expect(err).NotTo(HaveOccurred())
		perms, err := dutyRBAC.PermsForUser(uuid.Nil.String())
		Expect(err).NotTo(HaveOccurred())
		for _, p := range perms {
			Expect(slices.Contains(policy.AllObjects, p.Object)).To(BeTrue())
		}
		adminSpecificPerms := lo.Filter(perms, func(p policy.Permission, _ int) bool { return p.Subject != policy.RoleEveryone })
		Expect(len(adminSpecificPerms)).To(Equal(len(policy.AllObjects)))
	})
})
