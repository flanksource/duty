package models

import (
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var _ types.ResourceSelectable = Canary{}

type Canary struct {
	ID          uuid.UUID           `json:"id" yaml:"id" gorm:"default:generate_ulid()"`
	Name        string              `json:"name" yaml:"name"`
	Namespace   string              `json:"namespace" yaml:"namespace"`
	AgentID     uuid.UUID           `json:"agent_id" yaml:"agent_id"`
	Spec        types.JSON          `json:"spec" yaml:"spec"`
	Labels      types.JSONStringMap `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations types.JSONStringMap `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Source      string              `json:"source,omitempty" yaml:"source,omitempty"`
	Checks      types.JSONStringMap `gorm:"-" json:"checks,omitempty" yaml:"checks,omitempty"`
	CreatedAt   time.Time           `json:"created_at" yaml:"created_at" time_format:"postgres_timestamp" gorm:"<-:create"`
	UpdatedAt   *time.Time          `json:"updated_at" yaml:"updated_at" time_format:"postgres_timestamp" gorm:"autoUpdateTime:false"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty" time_format:"postgres_timestamp"`
}

func (t Canary) ConflictClause() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_id"}, {Name: "name"}, {Name: "namespace"}, {Name: "source"}},
		TargetWhere: clause.Where{
			Exprs: []clause.Expression{
				clause.And(
					clause.Eq{Column: "deleted_at"},
					clause.Expr{SQL: "agent_id = '00000000-0000-0000-0000-000000000000'::uuid", WithoutParentheses: true},
				),
			},
		},
		DoUpdates: clause.AssignmentColumns([]string{"labels", "spec", "annotations"}),
	}
}

func DeleteAllCanaries(db *gorm.DB, canaries ...Canary) error {
	ids := lo.Map(canaries, func(c Canary, _ int) string { return c.ID.String() })
	if err := db.Exec("DELETE FROM check_statuses WHERE check_id in (select id from checks where canary_id in (?))", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM check_config_relationships WHERE check_id in (select id from checks where canary_id in (?))", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM checks WHERE canary_id in (?)", ids).Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM canaries WHERE id in (?)", ids).Error; err != nil {
		return err
	}
	return nil
}

func (t Canary) GetUnpushed(db *gorm.DB) ([]DBTable, error) {
	var items []Canary
	err := db.Where("is_pushed IS FALSE").Find(&items).Error
	return lo.Map(items, func(i Canary, _ int) DBTable { return i }), err
}

func (c Canary) GetCheckID(checkName string) string {
	return c.Checks[checkName]
}

func (c Canary) PK() string {
	return c.ID.String()
}

func (c Canary) TableName() string {
	return "canaries"
}

func (c Canary) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

func (c Canary) GetID() string {
	return c.ID.String()
}

func (c Canary) GetName() string {
	return c.Name
}

func (c Canary) GetNamespace() string {
	return c.Namespace
}

func (c Canary) GetType() string {
	return ""
}

func (c Canary) GetAgentID() string {
	if c.AgentID == uuid.Nil {
		return ""
	}
	return c.AgentID.String()
}

func (c Canary) GetStatus() (string, error) {
	return "", nil
}

func (c Canary) GetHealth() (string, error) {
	return "", nil
}

func (c Canary) GetLabelsMatcher() labels.Labels {
	return canaryLabels{c}
}

func (c Canary) GetFieldsMatcher() fields.Fields {
	return noopMatcher{}
}

func DeleteChecksForCanary(db *gorm.DB, id string) ([]string, error) {
	var checkIDs []string
	var checks []Check
	err := db.Model(&checks).
		Table("checks").
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
		Where("canary_id = ? and deleted_at IS NULL", id).
		UpdateColumn("deleted_at", Now()).
		Error

	for _, c := range checks {
		checkIDs = append(checkIDs, c.ID.String())
	}
	return checkIDs, err
}

func DeleteCheckComponentRelationshipsForCanary(db *gorm.DB, id string) error {
	return db.Table("check_component_relationships").Where("canary_id = ?", id).UpdateColumn("deleted_at", Now()).Error
}

type canaryLabels struct {
	Canary
}

func (t canaryLabels) Has(field string) (exists bool) {
	if len(t.Labels) == 0 {
		return false
	}

	_, ok := (t.Labels)[field]
	return ok
}

func (t canaryLabels) Get(key string) (value string) {
	if len(t.Labels) == 0 {
		return ""
	}

	return (t.Labels)[key]
}
func (t canaryLabels) Lookup(key string) (value string, exists bool) {
	if len(t.Labels) == 0 {
		return "", false
	}
	value, ok := (t.Labels)[key]
	return value, ok
}
