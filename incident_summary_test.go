package duty

import (
	"context"
	"encoding/json"

	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/hack"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// actor represents a responder, a commenter or a incident commander
type actor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

var _ = ginkgo.Describe("Check incident_summary view", ginkgo.Ordered, func() {
	ginkgo.It("Should query incident_summary view", func() {
		row := hack.TestDBPGPool.QueryRow(context.Background(), "SELECT id, incident_id, title, responders, commenters, commander FROM incident_summary")
		var id, incidentID, title string
		var respondersRaw, commentersRaw, commanderRaw json.RawMessage

		err := row.Scan(&id, &incidentID, &title, &respondersRaw, &commentersRaw, &commanderRaw)
		Expect(err).ToNot(HaveOccurred())

		Expect(id).To(Equal(dummy.LogisticsAPIDownIncident.ID.String()))
		Expect(title).To(Equal(dummy.LogisticsAPIDownIncident.Title))

		Expect(incidentID).To(Equal("INC0000001"))

		var commander actor
		err = json.Unmarshal(commanderRaw, &commander)
		Expect(err).ToNot(HaveOccurred())
		Expect(commander.ID).To(Equal(dummy.JohnDoe.ID.String()))
		Expect(commander.Name).To(Equal(dummy.JohnDoe.Name))

		var responders []actor
		err = json.Unmarshal(respondersRaw, &responders)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(responders)).To(Equal(2))
		Expect(responders).To(Equal([]actor{
			{
				ID:     dummy.JohnDoe.ID.String(),
				Name:   dummy.JohnDoe.Name,
				Avatar: dummy.JohnDoe.Avatar,
			},
			{
				ID:     dummy.JohnWick.ID.String(),
				Name:   dummy.JohnWick.Name,
				Avatar: dummy.JohnWick.Avatar,
			},
		}))

		var commenters []actor
		err = json.Unmarshal(commentersRaw, &commenters)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(commenters)).To(Equal(2))
		Expect(commenters).To(Equal([]actor{
			{
				ID:     dummy.JohnDoe.ID.String(),
				Name:   dummy.JohnDoe.Name,
				Avatar: dummy.JohnDoe.Avatar,
			},
			{
				ID:     dummy.JohnWick.ID.String(),
				Name:   dummy.JohnWick.Name,
				Avatar: dummy.JohnWick.Avatar,
			},
		}))
	})
})
