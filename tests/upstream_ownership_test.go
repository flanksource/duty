package tests

import (
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/upstream"
)

var _ = ginkgo.Describe("agent ownership", ginkgo.Ordered, func() {
	homelab := dummy.HomelabAgent.ID
	gcp := dummy.GCPAgent.ID

	homelabConfig := models.ConfigItem{
		ID:          uuid.New(),
		AgentID:     homelab,
		Name:        lo.ToPtr("homelab-node"),
		Type:        lo.ToPtr("Kubernetes::Node"),
		ConfigClass: "Node",
		Config:      lo.ToPtr(`{"owner":"homelab"}`),
	}
	run := models.PlaybookRun{
		ID:         uuid.New(),
		PlaybookID: dummy.EchoConfig.ID,
		Status:     models.PlaybookRunStatusRunning,
		Spec:       []byte("{}"),
	}
	action := models.PlaybookRunAction{
		ID:            uuid.New(),
		Name:          "homelab-action",
		PlaybookRunID: run.ID,
		Status:        models.PlaybookActionStatusRunning,
		AgentID:       &homelab,
	}

	// push and remove do what the push handler does with a request from the agent.
	push := func(agentID uuid.UUID, req upstream.PushData) error {
		req.PopulateAgentID(agentID)
		return upstream.InsertUpstreamMsg(DefaultContext, agentID, &req)
	}
	remove := func(agentID uuid.UUID, req upstream.PushData) error {
		req.PopulateAgentID(agentID)
		return upstream.DeleteOnUpstream(DefaultContext, agentID, &req)
	}

	getConfig := func() models.ConfigItem {
		var c models.ConfigItem
		Expect(DefaultContext.DB().Where("id = ?", homelabConfig.ID).First(&c).Error).To(Succeed())
		return c
	}
	getAction := func() models.PlaybookRunAction {
		var a models.PlaybookRunAction
		Expect(DefaultContext.DB().Where("id = ?", action.ID).First(&a).Error).To(Succeed())
		return a
	}
	configUpdate := func() models.ConfigItem {
		c := homelabConfig
		c.Name = lo.ToPtr("renamed")
		c.Config = lo.ToPtr(`{"owner":"changed"}`)
		return c
	}
	failedResult := func() models.PlaybookRunAction {
		return models.PlaybookRunAction{ID: action.ID, Status: models.PlaybookActionStatusFailed, Error: lo.ToPtr("pushed failure")}
	}

	ginkgo.BeforeAll(func() {
		db := DefaultContext.DB()
		Expect(db.Create(&homelabConfig).Error).To(Succeed())
		Expect(db.Create(&run).Error).To(Succeed())
		Expect(db.Create(&action).Error).To(Succeed())
	})

	ginkgo.AfterAll(func() {
		db := DefaultContext.DB()
		Expect(db.Delete(&models.PlaybookRunAction{}, "id = ?", action.ID).Error).To(Succeed())
		Expect(db.Delete(&models.PlaybookRun{}, "id = ?", run.ID).Error).To(Succeed())
		Expect(db.Delete(&models.ConfigItem{}, "id = ?", homelabConfig.ID).Error).To(Succeed())
	})

	ginkgo.It("doesn't let an agent overwrite another agent's config item", func() {
		Expect(push(gcp, upstream.PushData{ConfigItems: []models.ConfigItem{configUpdate()}})).To(Succeed())

		c := getConfig()
		Expect(*c.Name).To(Equal("homelab-node"))
		Expect(*c.Config).To(MatchJSON(`{"owner":"homelab"}`))
		Expect(c.AgentID).To(Equal(homelab))
	})

	ginkgo.It("doesn't let an agent delete another agent's config item", func() {
		Expect(remove(gcp, upstream.PushData{ConfigItems: []models.ConfigItem{homelabConfig}})).To(Succeed())

		var count int64
		Expect(DefaultContext.DB().Model(&models.ConfigItem{}).Where("id = ?", homelabConfig.ID).Count(&count).Error).To(Succeed())
		Expect(count).To(Equal(int64(1)))
	})

	ginkgo.It("doesn't let an agent report the result of another agent's playbook action", func() {
		Expect(push(gcp, upstream.PushData{PlaybookActions: []models.PlaybookRunAction{failedResult()}})).To(Succeed())

		a := getAction()
		Expect(a.Status).To(Equal(models.PlaybookActionStatusRunning))
		Expect(a.Error).To(BeNil())
	})

	ginkgo.It("applies the owner's config update and action result", func() {
		Expect(push(homelab, upstream.PushData{
			ConfigItems:     []models.ConfigItem{configUpdate()},
			PlaybookActions: []models.PlaybookRunAction{failedResult()},
		})).To(Succeed())

		c := getConfig()
		Expect(*c.Name).To(Equal("renamed"))
		Expect(*c.Config).To(MatchJSON(`{"owner":"changed"}`))

		a := getAction()
		Expect(a.Status).To(Equal(models.PlaybookActionStatusFailed))
		Expect(a.Error).To(Equal(lo.ToPtr("pushed failure")))
	})

})
