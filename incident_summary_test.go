package duty

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/flanksource/duty/fixtures/dummy"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/testutils"
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
	Severity   string
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
		err := testutils.TestDB.Raw("SELECT * FROM incident_summary").Scan(&incidents).Error
		Expect(err).ToNot(HaveOccurred())

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
				responders = []actor{
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
				}
				commenters = []actor{
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
				}
			case dummy.UIDownIncident.ID:
				incident = dummy.UIDownIncident
				commander.FromPerson(dummy.JohnWick)
				responders = []actor{
					{
						ID:   dummy.BackendTeam.ID.String(),
						Name: dummy.BackendTeam.Name,
					},
				}
			default:
				ginkgo.Fail(fmt.Sprintf("unexpected incident: %s", incidentSummary.Title))
			}

			Expect(incidentSummary.ID).To(Equal(incident.ID))
			Expect(incidentSummary.IncidentID).To(BeElementOf([]string{"INC0000001", "INC0000002"}))
			Expect(incidentSummary.Title).To(Equal(incident.Title))
			Expect(incidentSummary.Severity).To(Equal(incident.Severity))
			Expect(incidentSummary.Type).To(Equal(incident.Type))
			Expect(incidentSummary.Status).To(Equal(incident.Status))
			Expect(incidentSummary.Commander).To(Equal(commander))
			Expect(incidentSummary.Responders).To(Equal(responders))
			Expect(incidentSummary.Commenters).To(Equal(commenters))
		}
	})
})
