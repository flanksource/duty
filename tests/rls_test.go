package tests

import (
	"fmt"

	"github.com/flanksource/duty/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type testCase struct {
	name          string
	jwtClaims     string
	expectedCount *int64
}

func verifyConfigCount(session *gorm.DB, jwtClaims string, expectedCount int64) {
	Expect(session.Exec(fmt.Sprintf("SET request.jwt.claims = '%s'", jwtClaims)).Error).To(BeNil())

	var count int64
	Expect(session.Model(&models.ConfigItem{}).Count(&count).Error).To(BeNil())
	Expect(count).To(Equal(expectedCount))
}

var _ = Describe("RLS test", Ordered, func() {
	var (
		tx                           *gorm.DB
		totalConfigs                 int64
		numConfigsWithFlanksourceTag int64
	)

	BeforeAll(func() {
		tx = DefaultContext.DB().Begin()

		Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Count(&totalConfigs).Error).To(BeNil())
		Expect(DefaultContext.DB().Where("tags->>'account' = 'flanksource'").Model(&models.ConfigItem{}).Count(&numConfigsWithFlanksourceTag).Error).To(BeNil())
	})

	AfterAll(func() {
		Expect(tx.Exec("RESET ROLE").Error).To(BeNil())
		Expect(tx.Commit().Error).To(BeNil())
	})

	for _, role := range []string{"postgrest_anon", "postgrest_api"} {
		Context(role, Ordered, func() {
			BeforeAll(func() {
				Expect(tx.Exec(fmt.Sprintf("SET ROLE '%s'", role)).Error).To(BeNil())

				var currentRole string
				Expect(tx.Raw("SELECT CURRENT_USER").Scan(&currentRole).Error).To(BeNil())
				Expect(currentRole).To(Equal(role))
			})

			DescribeTable("JWT claim tests",
				func(tc testCase) {
					verifyConfigCount(tx, tc.jwtClaims, *tc.expectedCount)
				},
				Entry("no permissions", testCase{
					name:          "no permissions",
					jwtClaims:     `{"tags": {"cluster": "testing-cluster"}, "agents": ["10000000-0000-0000-0000-000000000000"]}`,
					expectedCount: lo.ToPtr(int64(0)),
				}),
				Entry("correct agent", testCase{
					name:          "correct agent",
					jwtClaims:     `{"tags": {"cluster": "testing-cluster"}, "agents": ["00000000-0000-0000-0000-000000000000"]}`,
					expectedCount: &totalConfigs,
				}),
				Entry("correct tag", testCase{
					name:          "correct tag",
					jwtClaims:     `{"tags": {"account": "flanksource"}, "agents": ["10000000-0000-0000-0000-000000000000"]}`,
					expectedCount: &numConfigsWithFlanksourceTag,
				}),
			)
		})
	}
})
