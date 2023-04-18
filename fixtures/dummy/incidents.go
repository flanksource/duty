package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var LogisticsAPIDownIncident = models.Incident{
	ID:          uuid.MustParse("7c05a739-8a1c-4999-85f7-d93d03f32044"),
	Title:       "Logistics API is down",
	CreatedBy:   JohnDoe.ID,
	Type:        models.IncidentTypeAvailability,
	Status:      models.IncidentStatusOpen,
	Severity:    "Blocker",
	CommanderID: &JohnDoe.ID,
}

var AllDummyIncidents = []models.Incident{LogisticsAPIDownIncident}

var FirstComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnWick.ID,
	Comment:    "This is a comment",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var SecondComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnDoe.ID,
	Comment:    "A comment by John Doe",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var ThirdComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnDoe.ID,
	Comment:    "Another comment by John Doe",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var AllDummyComments = []models.Comment{FirstComment, SecondComment, ThirdComment}

var JiraResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Type:       "Jira",
	PersonID:   &JohnWick.ID,
	CreatedBy:  JohnWick.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var GitHubIssueResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Type:       "GithubIssue",
	PersonID:   &JohnDoe.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var SlackResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Type:       "Slack",
	PersonID:   &JohnDoe.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  time.Now(),
	UpdatedAt:  time.Now(),
}

var AllDummyResponders = []models.Responder{JiraResponder, GitHubIssueResponder, SlackResponder}
