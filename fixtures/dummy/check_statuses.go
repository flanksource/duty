package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
)

var t1 = currentTime.Add(-15 * time.Minute)
var t2 = currentTime.Add(-10 * time.Minute)
var t3 = currentTime.Add(-5 * time.Minute)

var LogisticsAPIHealthHTTPCheckStatus1 = models.CheckStatus{
	CheckID:   LogisticsAPIHealthHTTPCheck.ID,
	Duration:  100,
	Status:    true,
	CreatedAt: t1,
	Time:      t1.Format("2006-01-02 15:04:05"),
}

var LogisticsAPIHealthHTTPCheckStatus2 = models.CheckStatus{
	CheckID:   LogisticsAPIHealthHTTPCheck.ID,
	Duration:  100,
	Status:    true,
	CreatedAt: t2,
	Time:      t2.Format("2006-01-02 15:04:05"),
}

var LogisticsAPIHealthHTTPCheckStatus3 = models.CheckStatus{
	CheckID:   LogisticsAPIHealthHTTPCheck.ID,
	Duration:  100,
	Status:    true,
	CreatedAt: t3,
	Time:      t3.Format("2006-01-02 15:04:05"),
}

var LogisticsAPIHomeHTTPCheckStatus1 = models.CheckStatus{
	CheckID:   LogisticsAPIHomeHTTPCheck.ID,
	Duration:  100,
	Status:    true,
	CreatedAt: t1,
	Time:      t3.Format("2006-01-02 15:04:05"),
}

var LogisticsDBCheckStatus1 = models.CheckStatus{
	CheckID:   LogisticsDBCheck.ID,
	Duration:  50,
	Status:    false,
	CreatedAt: t1,
	Time:      t1.Format("2006-01-02 15:04:05"),
}

var AllDummyCheckStatuses = []models.CheckStatus{
	LogisticsAPIHealthHTTPCheckStatus1,
	LogisticsAPIHealthHTTPCheckStatus2,
	LogisticsAPIHealthHTTPCheckStatus3,
	LogisticsAPIHomeHTTPCheckStatus1,
	LogisticsDBCheckStatus1,
}
