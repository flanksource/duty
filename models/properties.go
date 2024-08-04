package models

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flanksource/commons/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AppProperty struct {
	Name      string     `json:"name,omitempty"`
	Value     string     `json:"value,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
	UpdatedAt time.Time  `json:"updated_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" time_format:"postgres_timestamp" gorm:"default:CURRENT_TIMESTAMP()"`
}

func (p AppProperty) TableName() string {
	return "properties"
}

func (p AppProperty) PK() string {
	return p.Name
}

func parsePropertiesFile(filename string) ([]AppProperty, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var props []AppProperty
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			logger.Warnf("invalid line: %s", line)
			continue
		}

		key := strings.TrimSpace(tokens[0])
		value := strings.TrimSpace(tokens[1])
		props = append(props, AppProperty{
			Name:  key,
			Value: value,
		})
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return props, nil
}

func SetPropertiesInDBFromFile(db *gorm.DB, filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}
	props, err := parsePropertiesFile(filename)
	if err != nil {
		return err
	}
	return SetProperties(db, props)
}

func SetProperties(db *gorm.DB, props []AppProperty) error {
	var values []string
	for _, p := range props {
		values = append(values, fmt.Sprintf("('%s', '%s')", p.Name, p.Value))
	}

	if len(values) == 0 {
		return nil
	}
	query := fmt.Sprintf("INSERT INTO properties (name, value) VALUES %s ON CONFLICT (name) DO UPDATE SET value = excluded.value", strings.Join(values, ","))
	return db.Exec(query).Error
}
