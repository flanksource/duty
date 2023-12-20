package dummy

import (
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

var UIDownIncident = models.Incident{
	ID:          uuid.MustParse("0c00b8a6-5bf8-42a4-98fe-2d39ddcb67cb"),
	Title:       "UI is down",
	CreatedBy:   JohnDoe.ID,
	Type:        models.IncidentTypeAvailability,
	Status:      models.IncidentStatusOpen,
	Severity:    "Blocker",
	CommanderID: &JohnWick.ID,
}

var AllDummyIncidents = []models.Incident{LogisticsAPIDownIncident, UIDownIncident}

var FirstComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnWick.ID,
	Comment:    "This is a comment",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var SecondComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnDoe.ID,
	Comment:    "A comment by John Doe",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var ThirdComment = models.Comment{
	ID:         uuid.New(),
	CreatedBy:  JohnDoe.ID,
	Comment:    "Another comment by John Doe",
	IncidentID: LogisticsAPIDownIncident.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var AllDummyComments = []models.Comment{FirstComment, SecondComment, ThirdComment}

var JiraResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Type:       "Jira",
	PersonID:   &JohnWick.ID,
	CreatedBy:  JohnWick.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var GitHubIssueResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: LogisticsAPIDownIncident.ID,
	Type:       "GithubIssue",
	PersonID:   &JohnDoe.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var SlackResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: UIDownIncident.ID,
	Type:       "Slack",
	TeamID:     &BackendTeam.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var MsPlannerResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: UIDownIncident.ID,
	Type:       "MSPlanner",
	PersonID:   &JohnWick.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var TelegramResponder = models.Responder{
	ID:         uuid.New(),
	IncidentID: UIDownIncident.ID,
	Type:       "Telegram",
	PersonID:   &JohnDoe.ID,
	CreatedBy:  JohnDoe.ID,
	CreatedAt:  CurrentTime,
	UpdatedAt:  CurrentTime,
}

var AllDummyResponders = []models.Responder{JiraResponder, GitHubIssueResponder, SlackResponder, MsPlannerResponder, TelegramResponder}

var BackendTeam = models.Team{
	ID:        uuid.New(),
	Name:      "Backend",
	Icon:      "backend",
	CreatedBy: JohnDoe.ID,
	CreatedAt: CurrentTime,
	UpdatedAt: CurrentTime,
}

var FrontendTeam = models.Team{
	ID:        uuid.New(),
	Name:      "Frontend",
	Icon:      "frontend",
	CreatedBy: JohnDoe.ID,
	CreatedAt: CurrentTime,
	UpdatedAt: CurrentTime,
}

var AllDummyTeams = []models.Team{BackendTeam, FrontendTeam}
