package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
)

func generateStatus(check models.Check, t time.Time, schedule time.Duration, count int, passingMod int) []models.CheckStatus {
	var statuses = []models.CheckStatus{}

	for i := 0; i < count; i++ {
		status := true
		if i%passingMod == 0 {
			status = false
		}
		statuses = append(statuses, models.CheckStatus{
			CheckID:   check.ID,
			Status:    status,
			CreatedAt: t,
			Duration:  (1 + i) * 20,
			Time:      t.Add(-time.Minute * time.Duration(i)).Format(time.DateTime),
		})
	}
	return statuses
}

func AllDummyCheckStatuses() []models.CheckStatus {
	statuses := append(
		generateStatus(LogisticsAPIHealthHTTPCheck, CurrentTime, time.Minute, 70, 5),
		generateStatus(DeletedCheck, CurrentTime, time.Minute, 1, 1)[0],
		generateStatus(DeletedCheckOld, *DeletedCheckOld.CreatedAt, time.Minute, 1, 1)[0],
		models.CheckStatus{
			CheckID:   LogisticsAPIHomeHTTPCheck.ID,
			Duration:  100,
			Status:    true,
			CreatedAt: CurrentTime.Add(-15 * time.Minute),
			Time:      CurrentTime.Add(-5 * time.Minute).Format(time.DateTime),
		},
		models.CheckStatus{
			CheckID:   LogisticsDBCheck.ID,
			Duration:  50,
			Status:    false,
			CreatedAt: CurrentTime.Add(-15 * time.Minute),
			Time:      CurrentTime.Add(-70 * time.Minute).Format(time.DateTime),
		},
	)

	statuses = append(statuses, generateStatus(DeletedCheck1h, CurrentTime.Add(-15*time.Minute), time.Minute, 1, 1)[0])
	statuses = append(statuses, generateStatus(DeletedCheck1h, CurrentTime.Add(-2*time.Hour), time.Minute, 10, 2)...)

	// Check statuses from 2022-01-01
	// not dervied from current time for consistency
	statuses = append(statuses, generateStatus(CartAPIHeathCheckAgent, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Minute*5, 1440, 5)...) // 1440 check statuses spanning 5 days

	return statuses
}
