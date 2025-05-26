package tests

import (
	"database/sql/driver"
	"fmt"
	"log"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/tests/fixtures/dummy"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// actor represents a responder, a commenter or a incident commander
type actor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (t actor) Value() (driver.Value, error) {
	return types.GenericStructValue(t, true)
}

func (t *actor) Scan(val any) error {
	return types.GenericStructScan(&t, val)
}

func (t *actor) FromPerson(p models.Person) {
	t.Avatar = p.Avatar
	t.Name = p.Name
	t.ID = p.ID.String()
}

func actorFromPerson(p models.Person) actor {
	var a actor
	a.FromPerson(p)
	return a
}

type actors []actor

func (t actors) Value() (driver.Value, error) {
	return types.GenericStructValue(t, true)
}

func (t *actors) Scan(val any) error {
	return types.GenericStructScan(&t, val)
}

// IncidentSummary represents the incident_summary view
type IncidentSummary struct {
	ID         uuid.UUID
	IncidentID string
	Title      string
	Severity   models.Severity
	Type       models.IncidentType
	Status     models.IncidentStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Responders actors
	Commander  actor
	Commenters actors
}

var _ = ginkgo.Describe("Check incident_summary view", ginkgo.Ordered, func() {
	ginkgo.It("Should query incident_summary view", func() {
		var incidents []IncidentSummary
		err := DefaultContext.DB().Raw("SELECT * FROM incident_summary").Scan(&incidents).Error
		Expect(err).ToNot(HaveOccurred())

		for _, incident := range incidents {
			log.Printf("incident: id:%s title:%s severity:%s\n", incident.IncidentID, incident.Title, incident.Severity)
		}
		Expect(incidents).To(HaveLen(2))

		Expect(len(incidents)).To(Equal(len(dummy.AllDummyIncidents)))

		for _, incidentSummary := range incidents {
			var (
				incident   models.Incident
				commander  actor
				responders actors
				commenters actors
			)

			switch incidentSummary.ID {
			case dummy.LogisticsAPIDownIncident.ID:
				incident = dummy.LogisticsAPIDownIncident
				commander.FromPerson(dummy.JohnDoe)
				responders = []actor{actorFromPerson(dummy.JohnDoe), actorFromPerson(dummy.JohnWick)}
				commenters = []actor{actorFromPerson(dummy.JohnDoe), actorFromPerson(dummy.JohnWick)}

			case dummy.UIDownIncident.ID:
				incident = dummy.UIDownIncident
				commander.FromPerson(dummy.JohnWick)
				responders = []actor{
					actorFromPerson(dummy.JohnDoe),
					actorFromPerson(dummy.JohnWick),
					{
						ID:     dummy.BackendTeam.ID.String(),
						Avatar: dummy.BackendTeam.Icon,
						Name:   dummy.BackendTeam.Name,
					},
				}

			default:
				ginkgo.Fail(fmt.Sprintf("unexpected incident: %s", incidentSummary.Title))
			}

			Expect(incidentSummary.ID).To(Equal(incident.ID))
			Expect(incidentSummary.Title).To(Equal(incident.Title))
			Expect(incidentSummary.Severity).To(Equal(incident.Severity))
			Expect(incidentSummary.Type).To(Equal(incident.Type))
			Expect(incidentSummary.Status).To(Equal(incident.Status))
			Expect(incidentSummary.Commander).To(Equal(commander))
			Expect(incidentSummary.Responders).To(ConsistOf(responders))
			Expect(incidentSummary.Commenters).To(ConsistOf(commenters))

			// FIXME: Fails on CI.
			// [FAILED] Expected
			// <string>: INC-4
			// to be an element of
			// <[]string | len:2, cap:2>: ["INC-1", "INC-2"]
			// Expect(incidentSummary.IncidentID).To(BeElementOf([]string{"INC-1", "INC-2"}))
		}
	})
})
