package tests

import (
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
)

var _ = ginkgo.Describe("access token roles", func() {
	roundTrip := func(roles models.AccessTokenRoles) models.AccessToken {
		token := models.AccessToken{
			ID:       uuid.New(),
			Name:     "roles-" + uuid.NewString(),
			Value:    uuid.NewString(),
			PersonID: dummy.JohnDoe.ID,
			Roles:    roles,
		}
		Expect(DefaultContext.DB().Create(&token).Error).ToNot(HaveOccurred())
		ginkgo.DeferCleanup(func() {
			Expect(DefaultContext.DB().Delete(&token).Error).ToNot(HaveOccurred())
		})

		var fetched models.AccessToken
		Expect(DefaultContext.DB().Where("id = ?", token.ID).First(&fetched).Error).ToNot(HaveOccurred())
		return fetched
	}

	storedRoles := func(id uuid.UUID) *string {
		var raw *string
		Expect(DefaultContext.DB().Raw("SELECT roles::text FROM access_tokens WHERE id = ?", id).Scan(&raw).Error).ToNot(HaveOccurred())
		return raw
	}

	ginkgo.It("round-trips a token with roles", func() {
		roles := models.AccessTokenRoles{
			{Namespace: "mission-control", Name: "viewer"},
			{Namespace: "default", Name: "payments-runner"},
		}
		fetched := roundTrip(roles)
		Expect(fetched.Roles).To(Equal(roles))
		Expect(storedRoles(fetched.ID)).To(HaveValue(MatchJSON(`[{"namespace":"mission-control","name":"viewer"},{"namespace":"default","name":"payments-runner"}]`)))
	})

	ginkgo.It("round-trips a token without roles as null", func() {
		fetched := roundTrip(nil)
		Expect(fetched.Roles).To(BeNil())
		Expect(storedRoles(fetched.ID)).To(BeNil())
	})

	ginkgo.It("stores empty roles as null", func() {
		fetched := roundTrip(models.AccessTokenRoles{})
		Expect(fetched.Roles).To(BeEmpty())
		Expect(storedRoles(fetched.ID)).To(BeNil())
	})
})
