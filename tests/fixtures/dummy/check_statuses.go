package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
)

var t1 = currentTime.Add(-15 * time.Minute)
var t3 = currentTime.Add(-5 * time.Minute)

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

var LogisticsAPIHomeHTTPCheckStatus1 = models.CheckStatus{
	CheckID:   LogisticsAPIHomeHTTPCheck.ID,
	Duration:  100,
	Status:    true,
	CreatedAt: t1,
	Time:      t3.Format(time.DateTime),
}

var OlderThan1H = models.CheckStatus{
	CheckID:   LogisticsDBCheck.ID,
	Duration:  50,
	Status:    false,
	CreatedAt: t1,
	Time:      time.Now().Add(-70 * time.Minute).Format(time.DateTime),
}

var AllDummyCheckStatuses = append(
	generateStatus(LogisticsAPIHealthHTTPCheck, time.Now(), 70, 5),
	generateStatus(DeletedCheck, time.Now(), 1, 1)[0],
	generateStatus(DeletedCheckOld, *DeletedCheckOld.CreatedAt, 1, 1)[0],
	LogisticsAPIHomeHTTPCheckStatus1,
	OlderThan1H)
