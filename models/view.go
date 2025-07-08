package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
	ID       uuid.UUID `json:"id" gorm:"primaryKey"`
	AgentID  uuid.UUID `json:"agent_id"`
	IsPushed bool      `json:"is_pushed" gorm:"default:false"`

	Results types.JSON `json:"results"`
}

func (ViewPanel) TableName() string {
	return "view_panels"
}

func (v ViewPanel) PK() string {
	return v.ID.String()
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
	ViewTableName string         `json:"view_table_name"`
	Row           map[string]any `json:"data"`
}

func (v GeneratedViewTable) PK() string {
	return fmt.Sprintf("%s", v.Row["id"]) // TODO: Must be fetched from a proper primary key column.
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

// UpdateIsPushed implements custom logic for updating is_pushed on dynamic view tables
func (GeneratedViewTable) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	tableGroups := make(map[string][]DBTable)
	viewIDsToUpdate := make(map[uuid.UUID]uuid.UUID) // map of view_id to agent_id

	for _, item := range items {
		if generatedViewTable, ok := item.(GeneratedViewTable); ok {
			tableGroups[generatedViewTable.TableName()] = append(tableGroups[generatedViewTable.TableName()], item)
		}
	}

	// Update each table
	for tableName, records := range tableGroups {
		if !db.Migrator().HasTable(tableName) {
			continue
		}

		// Build list of IDs to update
		var ids []any
		for _, record := range records {
			if viewData, ok := record.(GeneratedViewTable); ok {
				if id, exists := viewData.Row["id"]; exists {
					ids = append(ids, id)
				}
			}
		}

		if len(ids) > 0 {
			query := fmt.Sprintf("UPDATE %s SET is_pushed = true WHERE id IN (?)", tableName)
			if err := db.Exec(query, ids).Error; err != nil {
				return fmt.Errorf("failed to update is_pushed on %s: %w", tableName, err)
			}
		}
	}

	// Update the parent view's agent_id after successful reconciliation
	for viewID, agentID := range viewIDsToUpdate {
		if err := db.Model(&View{}).Where("id = ?", viewID).Update("agent_id", agentID).Error; err != nil {
			return fmt.Errorf("failed to update view agent_id for view %s: %w", viewID, err)
		}
	}

	return nil
}

// GetUnpushed returns all unpushed records from all view_* tables
func (t GeneratedViewTable) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	records, err := pkgDB.ReadTable(db, t.ViewTableName)
	if err != nil {
		return nil, err
	}

	var result []DBTable
	for _, record := range records {
		result = append(result, GeneratedViewTable{
			ViewTableName: t.ViewTableName,
			Row:           record,
		})
	}

	return result, nil
}
