package dummy

import "time"

var (
	DummyCreatedAt           = time.Date(2022, time.December, 31, 23, 59, 0, 0, time.UTC)
	DummyCreatedAtPlus3Years = time.Date(2025, time.December, 31, 23, 59, 0, 0, time.UTC)

	DummyNow         = time.Now()
	DummyYearOldDate = CurrentTime.AddDate(-1, 0, 0)
)
