package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var JohnDoe = models.Person{
	ID:    uuid.MustParse("01653e30-39a6-482a-8a9c-2bb8debaf440"),
	Name:  "John Doe",
	Email: "john@doe.com",
}

var JohnWick = models.Person{
	ID:    uuid.MustParse("3b6e2e89-b7ab-4751-a2d1-1e205fa478f6"),
	Name:  "John Wick",
	Email: "john@wick.com",
}

var AllDummyPeople = []models.Person{JohnDoe, JohnWick}
