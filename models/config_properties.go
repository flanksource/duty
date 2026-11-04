package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// ConfigProperty represents properties associated with config items
type ConfigProperty struct {
	ID        uuid.UUID `json:"id" gorm:"default:generate_ulid()"`
	ConfigID  uuid.UUID `json:"config_id"`
	ScraperID uuid.UUID `json:"scraper_id"`
	Label     string    `json:"label,omitempty"`
	Text      string    `json:"text,omitempty" gorm:"default:NULL"`
	Value     *float64  `json:"value,omitempty" gorm:"default:NULL"`
	Unit      string    `json:"unit,omitempty" gorm:"default:NULL"`
	Max       *int64    `json:"max,omitempty" gorm:"default:NULL"`
	Min       *int64    `json:"min,omitempty" gorm:"default:NULL"`
	CreatedAt time.Time `json:"created_at" gorm:"<-:create"`
	UpdatedAt time.Time `json:"updated_at"`

	// Visual properties
	Tooltip string `json:"tooltip,omitempty" gorm:"default:NULL"`
	Icon    string `json:"icon,omitempty" gorm:"default:NULL"`
	Type    string `json:"type,omitempty" gorm:"default:NULL"`
	Color   string `json:"color,omitempty" gorm:"default:NULL"`
	Order   int    `json:"order,omitempty" gorm:"default:NULL"`
}

func (ConfigProperty) TableName() string {
	return "config_properties"
}

func (p ConfigProperty) PK() string {
	return p.ID.String()
}

func (p ConfigProperty) PKCols() []clause.Column {
	return []clause.Column{{Name: "id"}}
}
