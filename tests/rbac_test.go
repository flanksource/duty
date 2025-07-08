package tests

import (
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"

	dutyRBAC "github.com/flanksource/duty/rbac"
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
})
