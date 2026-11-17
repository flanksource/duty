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
	homelabArtifact := models.Artifact{
		ID:           uuid.New(),
		ScraperID:    &dummy.HomelabKubeScraper.ID,
		ConnectionID: uuid.New(),
		Path:         "playbooks/" + action.ID.String() + "/out.txt",
		Filename:     "out.txt",
	}
	var pushedArtifacts []uuid.UUID

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
	getArtifact := func(id uuid.UUID) models.Artifact {
		var a models.Artifact
		Expect(DefaultContext.DB().Where("id = ?", id).First(&a).Error).To(Succeed())
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
		Expect(db.Create(&homelabArtifact).Error).To(Succeed())
	})

	ginkgo.AfterAll(func() {
		db := DefaultContext.DB()
		Expect(db.Delete(&models.Artifact{}, "id IN ?", append(pushedArtifacts, homelabArtifact.ID)).Error).To(Succeed())
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

	ginkgo.It("stores a pushed artifact under the agent's own path, ignoring the path it sent", func() {
		colliding := models.Artifact{ID: uuid.New(), CheckID: &dummy.CartAPIHeathCheckAgent.ID, ConnectionID: uuid.New(), Path: homelabArtifact.Path, Filename: "out.txt"}
		traversing := models.Artifact{ID: uuid.New(), CheckID: &dummy.CartAPIHeathCheckAgent.ID, ConnectionID: uuid.New(), Path: "../../etc/x", Filename: "x"}
		pushedArtifacts = append(pushedArtifacts, colliding.ID, traversing.ID)

		Expect(push(gcp, upstream.PushData{Artifacts: []models.Artifact{colliding, traversing}})).To(Succeed())

		Expect(getArtifact(colliding.ID).Path).To(Equal("agents/" + gcp.String() + "/" + colliding.ID.String()))
		Expect(getArtifact(traversing.ID).Path).To(Equal("agents/" + gcp.String() + "/" + traversing.ID.String()))
		Expect(getArtifact(homelabArtifact.ID).Path).To(Equal(homelabArtifact.Path))
	})

	ginkgo.It("keeps the path of an existing artifact when its owner pushes it again", func() {
		repushed := homelabArtifact
		repushed.Path = "somewhere/else"
		repushed.Filename = "renamed.txt"

		Expect(push(homelab, upstream.PushData{Artifacts: []models.Artifact{repushed}})).To(Succeed())

		a := getArtifact(homelabArtifact.ID)
		Expect(a.Path).To(Equal(homelabArtifact.Path))
		Expect(a.Filename).To(Equal("renamed.txt"))
	})

	ginkgo.It("doesn't let an agent overwrite another agent's artifact with the same ID", func() {
		overwrite := models.Artifact{ID: homelabArtifact.ID, CheckID: &dummy.CartAPIHeathCheckAgent.ID, ConnectionID: uuid.New(), Filename: "changed.txt"}

		Expect(push(gcp, upstream.PushData{Artifacts: []models.Artifact{overwrite}})).To(Succeed())

		a := getArtifact(homelabArtifact.ID)
		Expect(a.ScraperID).To(Equal(&dummy.HomelabKubeScraper.ID))
		Expect(a.CheckID).To(BeNil())
		Expect(a.Filename).To(Equal("renamed.txt"))
		Expect(a.Path).To(Equal(homelabArtifact.Path))
	})

	ginkgo.It("reports whether an artifact belongs to an agent", func() {
		owned, err := upstream.IsOwnedBy(DefaultContext, "artifacts", homelab, homelabArtifact.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(owned).To(BeTrue())

		owned, err = upstream.IsOwnedBy(DefaultContext, "artifacts", gcp, homelabArtifact.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(owned).To(BeFalse())

		owned, err = upstream.IsOwnedBy(DefaultContext, "artifacts", homelab, uuid.New())
		Expect(err).ToNot(HaveOccurred())
		Expect(owned).To(BeFalse())
	})
})
