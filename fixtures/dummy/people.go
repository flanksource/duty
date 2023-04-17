package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var JohnDoe = models.Person{
	ID:   uuid.New(),
	Name: "John Doe",
}

var JohnWick = models.Person{
	ID:   uuid.New(),
	Name: "John Wick",
}

var AllDummyPeople = []models.Person{JohnDoe, JohnWick}
