package dummy

import (
	"github.com/google/uuid"

	"github.com/flanksource/duty/models"
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

var AlanTuring = models.Person{
	ID:    uuid.MustParse("1603957c-72e9-4747-a2e1-9e9087c31b4e"),
	Name:  "Alan Turing",
	Email: "alan@turing.com",
}

var AllDummyPeople = []models.Person{JohnDoe, JohnWick, AlanTuring}
