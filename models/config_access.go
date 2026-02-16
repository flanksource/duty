package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"k8s.io/apimachinery/pkg/fields"
)

type ExternalUser struct {
	types.NoOpResourceSelectable

	ID        uuid.UUID      `json:"id" gorm:"default:generate_ulid()"`
	Aliases   pq.StringArray `json:"aliases,omitempty" gorm:"type:[]text"`
	Name      string         `json:"name"`
	AccountID string         `json:"account_id"`
	UserType  string         `json:"user_type"`
	Email     *string        `json:"email" gorm:"default:null"`
	ScraperID uuid.UUID      `json:"scraper_id" gorm:"not null"`
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

var _ types.ResourceSelectable = (*ExternalUser)(nil)

func (e ExternalUser) GetFieldsMatcher() fields.Fields {
	return types.GenericFieldMatcher{Fields: map[string]any{
		"id":         e.ID.String(),
		"name":       e.Name,
		"account_id": e.AccountID,
		"user_type":  e.UserType,
		"email":      e.Email,
		"scraper_id": e.ScraperID.String(),
	}}
}

func (e ExternalUser) GetID() string {
	return e.ID.String()
}

func (e ExternalUser) GetName() string {
	return e.Name
}

func (e ExternalUser) GetType() string {
	return e.UserType
}

type ExternalGroup struct {
	types.NoOpResourceSelectable

	ID        uuid.UUID      `json:"id"`
	ScraperID uuid.UUID      `json:"scraper_id" gorm:"not null"`
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

var _ types.ResourceSelectable = (*ExternalGroup)(nil)

func (e ExternalGroup) GetFieldsMatcher() fields.Fields {
	return types.GenericFieldMatcher{Fields: map[string]any{
		"id":         e.ID.String(),
		"name":       e.Name,
		"account_id": e.AccountID,
		"group_type": e.GroupType,
		"scraper_id": e.ScraperID.String(),
	}}
}

func (e ExternalGroup) GetID() string {
	return e.ID.String()
}

func (e ExternalGroup) GetName() string {
	return e.Name
}

func (e ExternalGroup) GetType() string {
	return e.GroupType
}

type ExternalUserGroup struct {
	ExternalUserID  uuid.UUID  `json:"external_user_id" gorm:"primaryKey"`
	ExternalGroupID uuid.UUID  `json:"external_group_id" gorm:"primaryKey"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletedBy       *uuid.UUID `json:"deleted_by"`
	CreatedAt       time.Time  `json:"created_at" gorm:"<-:create"`
	CreatedBy       *uuid.UUID `json:"created_by"`
}

func (e ExternalUserGroup) TableName() string {
	return "external_user_groups"
}

type ExternalRole struct {
	ID            uuid.UUID      `json:"id"`
	AccountID     string         `json:"account_id"`
	ApplicationID *uuid.UUID     `json:"application_id" gorm:"default:null"`
	ScraperID     *uuid.UUID     `json:"scraper_id" gorm:"default:null"`
	Aliases       pq.StringArray `json:"aliases" gorm:"type:[]text"`
	RoleType      string         `json:"role_type"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	CreatedAt     time.Time      `json:"created_at" gorm:"<-:create"`
	UpdatedAt     *time.Time     `json:"updated_at" gorm:"autoUpdateTime:false"`
	DeletedAt     *time.Time     `json:"deleted_at,omitempty"`
}

func (e ExternalRole) PK() string {
	return e.ID.String()
}

func (e ExternalRole) TableName() string {
	return "external_roles"
}

type AccessReview struct {
	ID              uuid.UUID      `json:"id"`
	ScraperID       uuid.UUID      `json:"scraper_id" gorm:"not null"`
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
	ID            string     `json:"id" gorm:"not null"`
	ApplicationID *uuid.UUID `json:"application_id" gorm:"default:null"`
	ScraperID     *uuid.UUID `json:"scraper_id" gorm:"default:null"`
	Source        *string    `json:"source" gorm:"default:null"`

	ConfigID        uuid.UUID  `json:"config_id"`
	ExternalUserID  *uuid.UUID `json:"external_user_id,omitempty"`
	ExternalGroupID *uuid.UUID `json:"external_group_id,omitempty"`
	ExternalRoleID  *uuid.UUID `json:"external_role_id,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	DeletedBy *uuid.UUID `json:"deleted_by,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`

	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	LastReviewedBy *uuid.UUID `json:"last_reviewed_by,omitempty"`
}

func (e ConfigAccess) TableName() string {
	return "config_access"
}

func (e ConfigAccess) PK() string {
	return e.ID
}

type ConfigAccessSummary struct {
	ConfigID        uuid.UUID  `json:"config_id"`
	ConfigName      string     `json:"config_name"`
	ConfigType      string     `json:"config_type"`
	ExternalGroupID *uuid.UUID `json:"external_group_id,omitempty"`
	ExternalUserID  uuid.UUID  `json:"external_user_id,omitempty"`
	Role            string     `json:"role"`
	User            string     `json:"user"`
	Email           string     `json:"email"`
	CreatedAt       time.Time  `json:"created_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	LastSignedInAt  *time.Time `json:"last_signed_in_at,omitempty"`
	LastReviewedAt  *time.Time `json:"last_reviewed_at,omitempty"`
	LastReviewedBy  *uuid.UUID `json:"last_reviewed_by,omitempty"`
}

func (e ConfigAccessSummary) TableName() string {
	return "config_access_summary"
}

type ConfigAccessLog struct {
	ConfigID       uuid.UUID     `json:"config_id" gorm:"primaryKey"`
	ExternalUserID uuid.UUID     `json:"external_user_id" gorm:"primaryKey"`
	ScraperID      uuid.UUID     `json:"scraper_id" gorm:"primaryKey"`
	CreatedAt      time.Time     `json:"created_at"`
	MFA            bool          `json:"mfa,omitempty" gorm:"default:null"`
	Properties     types.JSONMap `json:"properties,omitempty" gorm:"default:null"`
}

func (e ConfigAccessLog) TableName() string {
	return "config_access_logs"
}
