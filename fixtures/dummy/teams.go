package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var FrontendTeam = models.Team{
	ID:        uuid.New(),
	Name:      "Frontend",
	Icon:      "frontend",
	CreatedBy: JohnDoe.ID,
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
}

var BackendTeam = models.Team{
	ID:        uuid.MustParse("3d3f49ba-93d6-4058-8acc-96233f7c5c80"),
	Name:      "Backend",
	Spec:      []byte(`{"components": [{ "name": "logistics" }]}`),
	CreatedBy: JohnDoe.ID,
}

var PaymentTeam = models.Team{
	ID:        uuid.MustParse("72d965e2-b58b-4a23-ba73-2cae0daf5981"),
	Name:      "Payment",
	Spec:      []byte(`{"components": [{ "name": "logistics-ui" }]}`),
	CreatedBy: JohnDoe.ID,
}

var AllDummyTeams = []models.Team{BackendTeam, FrontendTeam, PaymentTeam}
