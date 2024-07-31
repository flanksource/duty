package tests

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = ginkgo.Describe("revoke agent access token when agent is deleted", ginkgo.Ordered, func() {
	var person *models.Person
	var agent *models.Agent
	var accessToken *models.AccessToken

	ginkgo.AfterAll(func() {
		DefaultContext.DB().Delete(accessToken)
		DefaultContext.DB().Delete(agent)
		DefaultContext.DB().Delete(person)
	})

	ginkgo.It("should create agent, person & access token", func() {
		person = &models.Person{
			ID:    uuid.New(),
			Name:  "test",
			Email: "random@email.com",
		}
		err := DefaultContext.DB().Create(person).Error
		Expect(err).ToNot(HaveOccurred())

		agent = &models.Agent{
			ID:       uuid.New(),
			Name:     "test",
			PersonID: &person.ID,
		}
		err = DefaultContext.DB().Create(agent).Error
		Expect(err).ToNot(HaveOccurred())

		accessToken = &models.AccessToken{
			ID:        uuid.New(),
			Name:      "test",
			PersonID:  person.ID,
			Value:     "dummy",
			CreatedAt: time.Now(),
			ExpiresAt: lo.ToPtr(time.Now().Add(365 * 24 * time.Hour)),
		}
		err = DefaultContext.DB().Create(accessToken).Error
		Expect(err).ToNot(HaveOccurred())
	})

	ginkgo.It("should revoke access token once agent is deleted", func() {
		err := DefaultContext.DB().Model(&models.Agent{}).Where("id = ?", agent.ID).UpdateColumn("deleted_at", "NOW()").Error
		Expect(err).ToNot(HaveOccurred())

		var deletedAgent models.Agent
		err = DefaultContext.DB().Where("id = ?", agent.ID).Find(&deletedAgent).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(deletedAgent.DeletedAt).To(Not(BeNil()))

		var activeAccessTokens []models.AccessToken
		err = DefaultContext.DB().Where("person_id = ?", person.ID).Where("expires_at > NOW()").Find(&activeAccessTokens).Error
		Expect(err).ToNot(HaveOccurred())
		Expect(len(activeAccessTokens)).To(BeZero())
	})
})
