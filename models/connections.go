package models

import (
	"net/url"
	"regexp"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

type Connection struct {
	ID          uuid.UUID           `gorm:"primaryKey;unique_index;not null;column:id" json:"id" faker:"uuid_hyphenated"  `
	Name        string              `gorm:"column:name" json:"name" faker:"name"  `
	Type        string              `gorm:"column:type" json:"type" faker:"oneof:  postgres, mysql, aws, gcp, http" `
	URL         string              `gorm:"column:url" json:"url,omitempty" faker:"url" template:"true"`
	Username    string              `gorm:"column:username" json:"username,omitempty" faker:"username"  `
	Password    string              `gorm:"column:password" json:"password,omitempty" faker:"password"  `
	Properties  types.JSONStringMap `gorm:"column:properties" json:"properties,omitempty" faker:"-"  `
	Certificate string              `gorm:"column:certificate" json:"certificate,omitempty" faker:"-"  `
	InsecureTLS bool                `gorm:"column:insecure_tls;default:false" json:"insecure_tls,omitempty" faker:"-"  `
	CreatedAt   time.Time           `gorm:"column:created_at;default:now()" json:"created_at,omitempty" faker:"-"  `
	UpdatedAt   time.Time           `gorm:"column:updated_at;default:now()" json:"updated_at,omitempty" faker:"-"  `
	CreatedBy   *uuid.UUID          `gorm:"column:created_by" json:"created_by,omitempty" faker:"-"  `
}

func (c Connection) String() string {
	if c.Type == "aws" {
		return "AWS::" + c.Username
	}
	var connection string
	// Obfuscate passwords of the form ' password=xxxxx ' from connectionString since
	// connectionStrings are used as metric labels and we don't want to leak passwords
	// Returns the Connection string with the password replaced by '###'
	if _url, err := url.Parse(c.URL); err == nil {
		if _url.User != nil {
			_url.User = nil
			connection = _url.String()
		}
	}
	//looking for a substring that starts with a space,
	//'password=', then any non-whitespace characters,
	//until an ending space
	re := regexp.MustCompile(`password=([^;]*)`)
	return re.ReplaceAllString(connection, "password=###")
}
