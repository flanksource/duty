package dummy

import "time"

var (
	DummyCreatedAt = time.Date(2022, time.December, 31, 23, 59, 0, 0, time.UTC)

	DummyYearOldDate = time.Now().AddDate(-1, 0, 0)
)

func ptr[T any](t T) *T {
	return &t
}
