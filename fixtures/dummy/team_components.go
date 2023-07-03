package dummy

import "github.com/flanksource/duty/models"

var LogisticBackendTeamComponent = models.TeamComponent{
	TeamID:      BackendTeam.ID,
	ComponentID: Logistics.ID,
	SelectorID:  ptr("366d4ecb71d8ce12cf253e55d541f987"),
}

var PaymentsTeamComponent = models.TeamComponent{
	TeamID:      PaymentTeam.ID,
	ComponentID: PaymentsAPI.ID,
	SelectorID:  ptr("7fbaeebb537818e8b334fd336613f8d4 "),
}

var AllTeamComponents = []models.TeamComponent{LogisticBackendTeamComponent, PaymentsTeamComponent}
