package models

import (
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"time"
)

// NotificationHealthState is the authoritative health marker, independent of event queue coalescing.
type NotificationHealthState struct {
	ResourceType string     `json:"resource_type"`
	ResourceID   uuid.UUID  `json:"resource_id"`
	Generation   uuid.UUID  `json:"generation"`
	EpisodeID    *uuid.UUID `json:"episode_id"`
	Health       string     `json:"health"`
	HealthySince *time.Time `json:"healthy_since"`
}

type NotificationHealthEpisode struct {
	ID           uuid.UUID  `json:"id"`
	ResourceType string     `json:"resource_type"`
	ResourceID   uuid.UUID  `json:"resource_id"`
	StartedAt    time.Time  `json:"started_at"`
	HealthyAt    *time.Time `json:"healthy_at"`
}

// NotificationDelivery contains routing snapshots, never credentials or transport URLs.
// It survives send history retention; each external operation has its own completion marker.
type NotificationDelivery struct {
	ID             uuid.UUID  `json:"id"`
	NotificationID uuid.UUID  `json:"notification_id"`
	EpisodeID      uuid.UUID  `json:"episode_id"`
	HistoryID      *uuid.UUID `json:"history_id"`
	ConnectionID   *uuid.UUID `json:"connection_id"`
	Transport      string     `json:"transport"`
	Destination    types.JSON `json:"destination"`
	Policy         types.JSON `json:"policy"`
	MessageID      string     `json:"message_id"`
	Channel        string     `json:"channel"`
	Status         string     `json:"status"`
	SentAt         *time.Time `json:"sent_at"`
	ReplyAt        *time.Time `json:"reply_at"`
	ReactionAt     *time.Time `json:"reaction_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	NotBefore      time.Time  `json:"not_before"`
	LeaseUntil     *time.Time `json:"lease_until"`
	LeaseToken     *uuid.UUID `json:"lease_token"`
	Attempts       int        `json:"attempts"`
	Error          *string    `json:"error"`
	CreatedAt      time.Time  `json:"created_at"`
}
