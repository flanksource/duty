package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
)

func generateStatus(check models.Check, t time.Time, count int, passingMod int) []models.CheckStatus {
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
			Time:      t.Add(time.Minute * time.Duration(i)).Format(time.DateTime),
		})
	}
	return statuses
}

func AllDummyCheckStatuses() []models.CheckStatus {
	statuses := append(
		generateStatus(LogisticsAPIHealthHTTPCheck, CurrentTime, 70, 5),
		generateStatus(DeletedCheck, CurrentTime, 1, 1)[0],
		generateStatus(DeletedCheckOld, *DeletedCheckOld.CreatedAt, 1, 1)[0],
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

	statuses = append(statuses, generateStatus(DeletedCheck1h, CurrentTime.Add(-2*time.Hour), 10, 2)...)

	return statuses
}
