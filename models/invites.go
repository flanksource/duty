package models

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID         uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	InvitedAt  time.Time  `json:"invited_at" gorm:"default:now()"`
	InvitedBy  *uuid.UUID `json:"invited_by,omitempty" gorm:"default:null"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty" gorm:"default:null"`
}

func (i Invite) TableName() string {
	return "invites"
}

func (i Invite) PK() string {
	return i.ID.String()
}
