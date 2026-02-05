package models

import (
	"time"
)

type ShortURL struct {
	Alias     string     `json:"alias" gorm:"primaryKey"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"  gorm:"<-:false"`
}

func (u ShortURL) PK() string {
	return u.Alias
}

func (u *ShortURL) TableName() string {
	return "short_urls"
}
