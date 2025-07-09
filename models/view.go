package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"

	pkgDB "github.com/flanksource/duty/db"
	"github.com/flanksource/duty/types"
)

// View represents the views database table
type View struct {
	ID        uuid.UUID  `json:"id" gorm:"default:generate_ulid()"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace" gorm:"default:NULL"`
	Spec      types.JSON `json:"spec"`
	Source    string     `json:"source" gorm:"default:KubernetesCRD"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"<-:create"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
	LastRan   *time.Time `json:"last_ran,omitempty" gorm:"default:NULL"`
	Error     *string    `json:"error,omitempty" gorm:"default:NULL"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func (v View) GeneratedTableName() string {
	cleanNamespace := strings.ReplaceAll(v.Namespace, "-", "_")
	cleanName := strings.ReplaceAll(v.Name, "-", "_")
	return fmt.Sprintf("view_%s_%s", cleanNamespace, cleanName)
}

func (v View) PK() string {
	return v.ID.String()
}

func (View) TableName() string {
	return "views"
}

func (v View) AsMap(removeFields ...string) map[string]any {
	return asMap(v, removeFields...)
}

func (v View) GetNamespace() string {
	return v.Namespace
}

// ViewPanel represents view panel data with push tracking
type ViewPanel struct {
	ViewID   uuid.UUID `json:"view_id" gorm:"primaryKey"`
	AgentID  uuid.UUID `json:"agent_id"`
	IsPushed bool      `json:"is_pushed" gorm:"default:false"`

	Results types.JSON `json:"results"`
}

func (ViewPanel) TableName() string {
	return "view_panels"
}

func (v ViewPanel) PK() string {
	return v.ViewID.String()
}

func (v ViewPanel) AsMap(removeFields ...string) map[string]any {
	return asMap(v, removeFields...)
}

// GetUnpushed returns all unpushed ViewPanel records
func (ViewPanel) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var records []ViewPanel
	if err := db.Where("is_pushed = ?", false).Find(&records).Error; err != nil {
		return nil, err
	}

	var result []DBTable
	for _, record := range records {
		result = append(result, record)
	}
	return result, nil
}

// GeneratedViewTable represents a record in a dynamically generated view_* table
type GeneratedViewTable struct {
	ViewTableName string                `json:"viewTableName"`
	PrimaryKey    []string              `json:"primaryKey"` // Column to use as the primary key
	Row           map[string]any        `json:"data"`
	ColumnDef     map[string]ColumnType `json:"columnDef"`
}

func (v GeneratedViewTable) PK() string {
	var keys []string
	for _, key := range v.PrimaryKey {
		if value, ok := v.Row[key]; ok {
			keys = append(keys, fmt.Sprintf("%s", value))
		}
	}

	return strings.Join(keys, "--")
}

func (v GeneratedViewTable) AsMap(removeFields ...string) map[string]any {
	return v.Row
}

func (v GeneratedViewTable) TableName() string {
	if v.ViewTableName != "" {
		return v.ViewTableName
	}

	return "generated_view_tables"
}

// GetUnpushed returns all unpushed records from all view_* tables
func (t GeneratedViewTable) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	records, err := pkgDB.ReadTable(db, t.ViewTableName, gorm.Expr("is_pushed = ?", false))
	if err != nil {
		return nil, err
	}

	if len(t.ColumnDef) > 0 {
		// Convert the values to native go types based on the column definition
		records = lo.Map(records, func(record map[string]any, _ int) map[string]any {
			return convertViewRecordsToNativeTypes(record, t.ColumnDef)
		})
	}

	var result []DBTable
	for _, record := range records {
		result = append(result, GeneratedViewTable{
			ViewTableName: t.ViewTableName,
			PrimaryKey:    t.PrimaryKey,
			Row:           record,
		})
	}

	return result, nil
}

func (t GeneratedViewTable) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	pks := lo.Map(items, func(item DBTable, _ int) string {
		return item.PK()
	})

	// FIXME: This doesn't work for composite primary keys.
	return db.Table(t.TableName()).
		Where(fmt.Sprintf("%s IN ?", strings.Join(t.PrimaryKey, ",")), pks).
		Update("is_pushed", true).Error
}
