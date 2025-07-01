package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"

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
	AgentID   uuid.UUID  `json:"agent_id"`
	IsPushed  bool       `json:"is_pushed" gorm:"default:false"`
	Error     *string    `json:"error,omitempty" gorm:"default:NULL"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
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

func (v View) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var records []View
	if err := db.Where("is_pushed = ?", false).Find(&records).Error; err != nil {
		return nil, err
	}

	var result []DBTable
	for _, record := range records {
		result = append(result, record)
	}
	return result, nil
}

type ViewPanel struct {
	ID       uuid.UUID `json:"id" gorm:"primaryKey"`
	AgentID  uuid.UUID `json:"agent_id"`
	IsPushed bool      `json:"is_pushed" gorm:"default:false"`

	// Results is a JSON array of panel results
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

func (v ViewPanel) UpdateParentsIsPushed(db *gorm.DB, items []DBTable) error {
	parentIDs := lo.Map(items, func(item DBTable, _ int) string {
		return item.(View).ID.String()
	})

	return db.Model(&View{}).Where("id IN ?", parentIDs).Update("is_pushed", false).Error
}

// GeneratedViewTable represents a record in a dynamically generated view_* table
type GeneratedViewTable struct {
	ViewTableName string         `json:"table_name"`
	Data          map[string]any `json:"data"`
}

func (v GeneratedViewTable) PK() string {
	return fmt.Sprintf("%s", v.Data["id"])
}

func (v GeneratedViewTable) AsMap(removeFields ...string) map[string]any {
	return v.Data
}

func (v GeneratedViewTable) TableName() string {
	return v.ViewTableName
}

// UpdateIsPushed implements custom logic for updating is_pushed on dynamic view tables
func (GeneratedViewTable) UpdateIsPushed(db *gorm.DB, items []DBTable) error {
	tableGroups := make(map[string][]DBTable)
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
				if id, exists := viewData.Data["id"]; exists {
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

	return nil
}

// GetUnpushed returns all unpushed records from all view_* tables
func (GeneratedViewTable) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	// First, get all views to determine which tables to check
	var views []View
	if err := db.Find(&views).Error; err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}

	var result []DBTable
	for _, view := range views {
		tableName := view.GeneratedTableName()
		if !db.Migrator().HasTable(tableName) {
			continue
		}

		// Query unpushed records from this view table
		rows, err := db.Raw(fmt.Sprintf("SELECT * FROM %s WHERE is_pushed = false", tableName)).Rows()
		if err != nil {
			continue // Skip if table doesn't have is_pushed column or other issues
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			continue
		}

		for rows.Next() {
			// Create a slice of any to hold the values
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}

			// Convert to map
			data := make(map[string]any)
			for i, col := range columns {
				data[col] = values[i]
			}

			result = append(result, GeneratedViewTable{
				ViewTableName: tableName,
				Data:          data,
			})
		}
	}

	return result, nil
}

func (v View) GeneratedTableName() string {
	cleanNamespace := strings.ReplaceAll(v.Namespace, "-", "_")
	cleanName := strings.ReplaceAll(v.Name, "-", "_")
	return fmt.Sprintf("view_%s_%s", cleanNamespace, cleanName)
}
