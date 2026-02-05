package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	pkgDB "github.com/flanksource/duty/db"
	"github.com/flanksource/duty/types"
)

// Ensure interface compliance.
//
// NOTE: A view isn't entirely resource selectable
// but we need it to implement the interface
// Casbin, which uses matchResourceSelector() func.
var _ types.ResourceSelectable = View{}

// View represents the views database table
type View struct {
	ID        uuid.UUID           `json:"id" gorm:"default:generate_ulid()"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace" gorm:"default:NULL"`
	Spec      types.JSON          `json:"spec"`
	Source    string              `json:"source" gorm:"default:KubernetesCRD"`
	Labels    types.JSONStringMap `json:"labels,omitempty" gorm:"type:jsonb"`
	CreatedBy *uuid.UUID          `json:"created_by,omitempty"`
	CreatedAt time.Time           `json:"created_at" gorm:"<-:create"`
	UpdatedAt *time.Time          `json:"updated_at" gorm:"autoUpdateTime:false"`
	Error     *string             `json:"error,omitempty" gorm:"default:NULL"`
	DeletedAt *time.Time          `json:"deleted_at,omitempty"`
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

// ResourceSelectable interface implementation for View
// Views only support namespace and name matching
func (v View) GetFieldsMatcher() fields.Fields {
	return noopMatcher{}
}

func (v View) GetLabelsMatcher() labels.Labels {
	return noopMatcher{}
}

func (v View) GetID() string {
	return v.ID.String()
}

func (v View) GetName() string {
	return v.Name
}

func (v View) GetType() string {
	return ""
}

func (v View) GetStatus() (string, error) {
	return "", nil
}

func (v View) GetHealth() (string, error) {
	return "", nil
}

// ViewPanel represents view panel data with push tracking
type ViewPanel struct {
	ViewID             uuid.UUID `json:"view_id" gorm:"primaryKey"`
	RequestFingerprint string    `json:"request_fingerprint" gorm:"primaryKey;default:''"`
	AgentID            uuid.UUID `json:"agent_id"`
	IsPushed           bool      `json:"is_pushed" gorm:"default:false"`

	// RefreshedAt is the last time this view was refreshed for this request fingerprint.
	//
	// NOTE: This is the refresh time of entire view (not just the panels. It also indicates the refresh time of the table)
	RefreshedAt *time.Time `json:"refreshed_at,omitempty" gorm:"default:now()"`

	// Results is a JSON array of panel results
	Results types.JSON `json:"results"`
}

func (ViewPanel) TableName() string {
	return "view_panels"
}

func (v ViewPanel) PK() string {
	// Composite key: ViewID:RequestFingerprint
	if v.RequestFingerprint == "" {
		return v.ViewID.String()
	}
	return fmt.Sprintf("%s:%s", v.ViewID.String(), v.RequestFingerprint)
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

func (t ViewPanel) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	// Build composite key conditions for ViewPanel with (view_id, request_fingerprint)
	var conditions [][]any
	for _, item := range items {
		c := any(item).(ViewPanel)
		conditions = append(conditions, []any{c.ViewID, c.RequestFingerprint})
	}

	if len(conditions) == 0 {
		return nil
	}

	// Use composite key in WHERE clause
	return db.Table(t.TableName()).
		Where("(view_id, request_fingerprint) IN ?", conditions).
		Update("is_pushed", true).Error
}

// GeneratedViewTable represents a record in a dynamically generated view_* table
type GeneratedViewTable struct {
	ViewTableName string                `json:"viewTableName"`
	PrimaryKey    []string              `json:"primaryKey"` // Columns to use as the primary key
	Row           map[string]any        `json:"data"`
	ColumnDef     map[string]ColumnType `json:"columnDef"`
}

func (v GeneratedViewTable) PK() string {
	// PK() is used to update is_pushed for the table.
	// This interface isn't suitable for composite primary keys.
	// GeneratedViewTable defines its own custom UpdateIsPushed method.
	return "not-implemented"
}

func (v GeneratedViewTable) AsMap(removeFields ...string) map[string]any {
	return v.Row
}

func (v GeneratedViewTable) TableName() string {
	return v.ViewTableName
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
			record, _ = ConvertRowToNativeTypes(record, t.ColumnDef)
			return record
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
	if len(t.PrimaryKey) == 0 {
		return fmt.Errorf("cannot update is_pushed for table: %s, primary key is empty", t.TableName())
	}

	pks := lo.Map(items, func(item DBTable, _ int) []any {
		c := any(item).(GeneratedViewTable)
		var pk []any
		for _, key := range t.PrimaryKey {
			if value, ok := c.Row[key]; ok {
				pk = append(pk, value)
			} else {
				pk = append(pk, nil)
			}
		}

		return pk
	})

	return db.Table(t.TableName()).
		Where(fmt.Sprintf("(%s) IN ?", strings.Join(t.PrimaryKey, ",")), pks).
		Update("is_pushed", true).Error
}
