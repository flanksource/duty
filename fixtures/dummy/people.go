package dummy

import (
	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var JohnDoe = models.Person{
	ID:   uuid.New(),
	Name: "John Doe",
}

var AllDummyPeople = []models.Person{JohnDoe}
