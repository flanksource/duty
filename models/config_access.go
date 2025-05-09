package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ExternalUser struct {
	ID        uuid.UUID  `json:"id"`
	Aliases   []string   `json:"aliases"`
	Name      string     `json:"name"`
	AccountID string     `json:"account_id"`
	UserType  string     `json:"user_type"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:create"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by"`
}

func (e ExternalUser) PK() string {
	return e.ID.String()
}

func (e ExternalUser) TableName() string {
	return "external_users"
}

type ExternalGroup struct {
	ID        uuid.UUID `json:"id"`
	AccountID string    `json:"account_id"`
	Aliases   []string  `json:"aliases"`
	Name      string    `json:"name"`
	GroupType string    `json:"group_type"`
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
	ID          uuid.UUID       `json:"id"`
	AccountID   string          `json:"account_id"`
	Aliases     []string        `json:"aliases"`
	RoleType    string          `json:"role_type"`
	Name        string          `json:"name"`
	Spec        json.RawMessage `json:"spec"`
	Description string          `json:"description"`
}

func (e ExternalRole) PK() string {
	return e.ID.String()
}

func (e ExternalRole) TableName() string {
	return "external_roles"
}

type AccessReview struct {
	ID              uuid.UUID  `json:"id"`
	Aliases         []string   `json:"aliases"`
	ConfigID        uuid.UUID  `json:"config_id"`
	ExternalGroupID *uuid.UUID `json:"external_group_id"`
	ExternalUserID  *uuid.UUID `json:"external_user_id"`
	ExternalRoleID  uuid.UUID  `json:"external_role_id"`
	CreatedAt       time.Time  `json:"created_at" gorm:"<-:create"`
	CreatedBy       *uuid.UUID `json:"created_by"`
	Source          string     `json:"source"`
}

func (e AccessReview) PK() string {
	return e.ID.String()
}

func (e AccessReview) TableName() string {
	return "access_reviews"
}

type ConfigAccess struct {
	ConfigID       uuid.UUID  `json:"config_id"`
	ExternalUser   *uuid.UUID `json:"external_user_id"`
	ExternalGroup  *uuid.UUID `json:"external_group_id"`
	ExternalRole   *uuid.UUID `json:"external_role_id"`
	CreatedAt      time.Time  `json:"created_at" gorm:"<-:create"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	DeletedBy      *uuid.UUID `json:"deleted_by"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	LastReviewedBy *uuid.UUID `json:"last_reviewed_by"`
	CreatedBy      *uuid.UUID `json:"created_by"`
}

func (e ConfigAccess) TableName() string {
	return "config_access"
}

func (e ConfigAccess) PK() string {
	return e.ConfigID.String()
}
