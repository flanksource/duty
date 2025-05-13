package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ExternalUser struct {
	ID        uuid.UUID      `json:"id" gorm:"default:generate_ulid()"`
	Aliases   pq.StringArray `json:"aliases,omitempty" gorm:"type:[]text"`
	Name      string         `json:"name"`
	AccountID string         `json:"account_id"`
	UserType  string         `json:"user_type"`
	Email     *string        `json:"email" gorm:"default:null"`
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt *time.Time     `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
	CreatedBy *string        `json:"created_by,omitempty" gorm:"default:null"`
}

func (e ExternalUser) PK() string {
	return e.ID.String()
}

func (e ExternalUser) TableName() string {
	return "external_users"
}

type ExternalGroup struct {
	ID        uuid.UUID      `json:"id"`
	AccountID string         `json:"account_id"`
	Aliases   pq.StringArray `json:"aliases,omitempty" gorm:"type:[]text"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt *time.Time     `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
	GroupType string         `json:"group_type"`
}

func (e ExternalGroup) PK() string {
	return e.ID.String()
}

func (e ExternalGroup) TableName() string {
	return "external_groups"
}

type ExternalUserGroup struct {
	ID              uuid.UUID  `json:"id"`
	ExternalUserID  uuid.UUID  `json:"external_user_id"`
	ExternalGroupID uuid.UUID  `json:"external_group_id"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletedBy       *uuid.UUID `json:"deleted_by"`
	CreatedAt       time.Time  `json:"created_at" gorm:"<-:create"`
	CreatedBy       *uuid.UUID `json:"created_by"`
}

func (e ExternalUserGroup) PK() string {
	return e.ID.String()
}

func (e ExternalUserGroup) TableName() string {
	return "external_user_groups"
}

type ExternalRole struct {
	ID          uuid.UUID      `json:"id"`
	AccountID   string         `json:"account_id"`
	Aliases     pq.StringArray `json:"aliases" gorm:"type:[]text"`
	RoleType    string         `json:"role_type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

func (e ExternalRole) PK() string {
	return e.ID.String()
}

func (e ExternalRole) TableName() string {
	return "external_roles"
}

type AccessReview struct {
	ID              uuid.UUID      `json:"id"`
	Aliases         pq.StringArray `json:"aliases" gorm:"type:[]text"`
	ConfigID        uuid.UUID      `json:"config_id"`
	ExternalGroupID *uuid.UUID     `json:"external_group_id"`
	ExternalUserID  *uuid.UUID     `json:"external_user_id"`
	ExternalRoleID  uuid.UUID      `json:"external_role_id"`
	CreatedAt       time.Time      `json:"created_at" gorm:"<-:create"`
	CreatedBy       *uuid.UUID     `json:"created_by"`
	Source          string         `json:"source"`
}

func (e AccessReview) PK() string {
	return e.ID.String()
}

func (e AccessReview) TableName() string {
	return "access_reviews"
}

type ConfigAccess struct {
	ID string `json:"id" gorm:"not null"`

	ConfigID        uuid.UUID  `json:"config_id"`
	ExternalUserID  *uuid.UUID `json:"external_user_id,omitempty"`
	ExternalGroupID *uuid.UUID `json:"external_group_id,omitempty"`
	ExternalRoleID  *uuid.UUID `json:"external_role_id,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	DeletedBy *uuid.UUID `json:"deleted_by,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`

	LastSignedInAt *time.Time `json:"last_signed_in_at,omitempty"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	LastReviewedBy *uuid.UUID `json:"last_reviewed_by,omitempty"`
}

func (e ConfigAccess) TableName() string {
	return "config_access"
}

func (e ConfigAccess) PK() string {
	return e.ID
}
