package dummy

import (
	"time"

	"github.com/flanksource/duty/models"
	"github.com/google/uuid"
)

var NoMatchNotification = models.Notification{
	ID:        uuid.MustParse("605dcaaa-9637-4ff3-bb57-233b8a151e3a"),
	Name:      "no-match-notification",
	Namespace: "default",
	CreatedAt: time.Now(),
	UpdatedAt: time.Now(),
	Title:     "The title",
	Template:  "The body",
	Filter:    "false", // matches nothing
	PersonID:  &JohnDoe.ID,
	Source:    models.SourceUI,
	Events:    []string{"config.unhealthy", "config.warn"},
}

var AllDummyNotifications = []models.Notification{NoMatchNotification}
